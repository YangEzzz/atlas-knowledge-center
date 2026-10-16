// setup-auth provisions Atlas without changing Nexus data or existing Helios credentials.
package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func connect(ctx context.Context, env map[string]string, name string) (*pgx.Conn, error) {
	cfg, err := pgx.ParseConfig("")
	if err != nil {
		return nil, err
	}
	cfg.Host = env["DB_HOST"]
	var port uint16
	if _, err = fmt.Sscan(env["DB_PORT"], &port); err != nil {
		return nil, err
	}
	cfg.Port = port
	cfg.User = env["DB_USER"]
	cfg.Password = env["DB_PASSWORD"]
	if cfg.Password == "" {
		cfg.Password = env["DB_PASS"]
	}
	cfg.Database = name
	cfg.ConnectTimeout = 10 * time.Second
	if env["DB_SSLMODE"] == "disable" {
		cfg.TLSConfig = nil
		cfg.Fallbacks = nil
	}
	return pgx.ConnectConfig(ctx, cfg)
}

func run() error {
	memberEmail := flag.String("member-email", "", "Explicitly authorized Helios member email")
	memberRole := flag.String("member-role", "", "Explicitly authorized project role")
	verifyEmail := flag.String("verify-member-email", "", "Read-only verification against the running Helios API")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		return err
	}
	nexus, err := godotenv.Read(filepath.Join(root, "..", "nexus-multilingual-backend", ".env"))
	if err != nil {
		return err
	}
	helios, err := godotenv.Read(filepath.Join(root, "..", "helios-auth-service", ".env"))
	if err != nil {
		return err
	}
	localPath := filepath.Join(root, "apps", "api", ".env")
	local, err := godotenv.Read(localPath)
	if err != nil {
		return err
	}
	if *verifyEmail != "" {
		return verifyMember(ctx, helios, local, *verifyEmail)
	}
	db, err := connect(ctx, nexus, nexus["DB_NAME"])
	if err != nil {
		return fmt.Errorf("Nexus database connection failed: %w", err)
	}
	defer db.Close(context.Background())
	var exists bool
	if err = db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", "atlas_db").Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err = db.Exec(ctx, `CREATE DATABASE atlas_db`); err != nil {
			return err
		}
	}
	fmt.Println("Atlas database ready: atlas_db (Nexus database untouched)")
	authDB, err := connect(ctx, helios, helios["DB_NAME"])
	if err != nil {
		return fmt.Errorf("Helios database connection failed: %w", err)
	}
	defer authDB.Close(context.Background())
	tx, err := authDB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	const project = "atlas-knowledge-center"
	var projectID string
	err = tx.QueryRow(ctx, `INSERT INTO projects(id, project_name, project_id_string, project_url, description, role_documentation, created_at, updated_at)
 VALUES(uuid_generate_v4(), 'Atlas 研发知识中心', $1, 'http://localhost:5175', '研发项目入口与知识门户', '{}', now(), now())
 ON CONFLICT(project_id_string) DO NOTHING RETURNING id`, project).Scan(&projectID)
	created := err == nil
	if err == pgx.ErrNoRows {
		err = tx.QueryRow(ctx, "SELECT id FROM projects WHERE project_id_string=$1", project).Scan(&projectID)
	}
	if err != nil {
		return err
	}
	for _, role := range []string{"admin", "productor", "developer"} {
		if _, err = tx.Exec(ctx, `INSERT INTO project_role_templates(id,project_id,role_name,description,created_at,updated_at)
  VALUES(uuid_generate_v4(),$1,$2,$3,now(),now()) ON CONFLICT(project_id,role_name) DO NOTHING`, projectID, role, role); err != nil {
			return err
		}
	}
	var storedHash, existingProject string
	err = tx.QueryRow(ctx, "SELECT secret_hash, project_id::text FROM sso_clients WHERE id=$1", project).Scan(&storedHash, &existingProject)
	secret := local["HELIOS_CLIENT_SECRET"]
	clientCreated := false
	if err == pgx.ErrNoRows {
		bytes := make([]byte, 32)
		if _, err = rand.Read(bytes); err != nil {
			return err
		}
		secret = base64.RawURLEncoding.EncodeToString(bytes)
		hash := sha256.Sum256([]byte(secret))
		if _, err = tx.Exec(ctx, `INSERT INTO sso_clients(id,project_id,name,secret_hash,redirect_uris,enabled,created_at,updated_at)
  VALUES($1,$2,'Atlas 研发知识中心',$3,$4,true,now(),now())`, project, projectID, hex.EncodeToString(hash[:]), []string{local["HELIOS_REDIRECT_URI"]}); err != nil {
			return err
		}
		clientCreated = true
	} else if err != nil {
		return err
	} else {
		hash := sha256.Sum256([]byte(secret))
		if existingProject != projectID || secret == "" || hex.EncodeToString(hash[:]) != storedHash {
			return fmt.Errorf("existing Atlas SSO credential does not match local configuration; refusing to rotate it")
		}
		if _, err = tx.Exec(ctx, `UPDATE sso_clients SET redirect_uris=array_append(redirect_uris,$2),updated_at=now() WHERE id=$1 AND NOT ($2=ANY(redirect_uris))`, project, local["HELIOS_REDIRECT_URI"]); err != nil {
			return err
		}
	}
	if *memberEmail != "" {
		if *memberRole != "admin" && *memberRole != "developer" && *memberRole != "productor" {
			return fmt.Errorf("unsupported project role")
		}
		var userID string
		if err = tx.QueryRow(ctx, "SELECT id FROM users WHERE lower(email)=lower($1) AND status='active'", *memberEmail).Scan(&userID); err != nil {
			return fmt.Errorf("authorized member not found or inactive: %w", err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO project_memberships(id,user_id,project_id,role_in_project,created_at) VALUES(uuid_generate_v4(),$1,$2,$3,now()) ON CONFLICT(user_id,project_id) DO UPDATE SET role_in_project=EXCLUDED.role_in_project`, userID, projectID, *memberRole); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO audit_logs(id,action,resource,details,ip_address,created_at) VALUES(uuid_generate_v4(),'add_member',$1,$2,'',now())`, "project:"+projectID, "Explicitly authorized Atlas membership: "+*memberEmail+" role="+*memberRole); err != nil {
			return err
		}
	}
	if created || clientCreated {
		if _, err = tx.Exec(ctx, `INSERT INTO audit_logs(id,action,resource,details,ip_address,created_at)
  VALUES(uuid_generate_v4(),'atlas_auth_setup',$1,'Provisioned Atlas project and SSO login configuration; no memberships changed','',now())`, "project:"+projectID); err != nil {
			return err
		}
	}
	// Store the new secret before committing. A retry can create a fresh unused secret
	// if the transaction fails. Never print secrets or put them in tracked files.
	local["HELIOS_CLIENT_SECRET"] = secret
	local["HELIOS_CLIENT_ID"] = project
	if err = godotenv.Write(local, localPath); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	var memberCount int
	if err = authDB.QueryRow(ctx, "SELECT count(*) FROM project_memberships WHERE project_id=$1", projectID).Scan(&memberCount); err != nil {
		return err
	}
	fmt.Printf("Helios project ready: %s (%s), SSO configured, members=%d\n", project, projectID, memberCount)
	return nil
}

func verifyMember(ctx context.Context, helios, local map[string]string, email string) error {
	db, err := connect(ctx, helios, helios["DB_NAME"])
	if err != nil {
		return err
	}
	defer db.Close(context.Background())
	var userID string
	if err = db.QueryRow(ctx, "SELECT id FROM users WHERE lower(email)=lower($1)", email).Scan(&userID); err != nil {
		return err
	}
	body, _ := json.Marshal(map[string]string{"user_id": userID})
	req, err := http.NewRequestWithContext(ctx, "POST", local["HELIOS_INTERNAL_URL"]+"/api/sso/identity", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(local["HELIOS_CLIENT_ID"], local["HELIOS_CLIENT_SECRET"])
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var result struct {
		Code int `json:"code"`
		Data struct {
			Project string `json:"project_id_string"`
			Role    string `json:"role_in_project"`
		} `json:"data"`
	}
	if err = json.NewDecoder(io.LimitReader(resp.Body, 65536)).Decode(&result); err != nil {
		return err
	}
	if resp.StatusCode != 200 || result.Code != 200 || result.Data.Project != local["HELIOS_PROJECT_ID"] || result.Data.Role != "admin" {
		return fmt.Errorf("Helios identity verification failed, HTTP %d", resp.StatusCode)
	}
	fmt.Println("Running Helios API verified Atlas client credentials and the requested admin membership")
	return nil
}

func main() {
	if err := run(); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}
