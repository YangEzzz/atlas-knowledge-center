<script setup lang="ts">
import { Skeleton } from '@/components/ui/skeleton'
</script>

<template>
  <div class="detail-page-skeleton" aria-label="加载中">
    <section class="document-summary-strip">
      <div class="detail-skeleton-main">
        <div class="document-meta-row">
          <Skeleton class="skeleton-pill" />
          <Skeleton class="skeleton-pill short" />
          <Skeleton class="skeleton-pill short" />
        </div>
        <Skeleton class="skeleton-line medium" />
      </div>
      <div class="skeleton-meta-grid">
        <Skeleton class="skeleton-meta-box" />
        <Skeleton class="skeleton-meta-box" />
        <Skeleton class="skeleton-meta-box hide-on-small" />
      </div>
    </section>

    <div class="document-detail-layout">
      <article class="document-reader-card">
        <div class="preview-document-body document-reader-body">
          <Skeleton class="skeleton-heading" />
          <Skeleton class="skeleton-line" />
          <Skeleton class="skeleton-line almost" />
          <Skeleton class="skeleton-line medium" />
          <div class="skeleton-block-space" />
          <Skeleton class="skeleton-subheading" />
          <Skeleton class="skeleton-line" />
          <Skeleton class="skeleton-line almost" />
          <Skeleton class="skeleton-line short-line" />
        </div>
      </article>

      <aside class="document-aside">
        <div class="skeleton-side-card">
          <Skeleton class="skeleton-line medium" />
          <Skeleton class="skeleton-line" />
          <Skeleton class="skeleton-line almost" />
          <Skeleton class="skeleton-line short-line" />
        </div>
        <div class="skeleton-side-card">
          <Skeleton class="skeleton-line medium" />
          <Skeleton class="skeleton-line almost" />
          <Skeleton class="skeleton-line short-line" />
        </div>
      </aside>
    </div>
  </div>
</template>
