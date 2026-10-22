import { DomEditor, type IDomEditor, type Toolbar } from "@wangeditor/editor";

// wangEditor 5 removes its editor mapping on destroy, but does not cancel the
// throttled selectionchange callback. A queued callback can then access a dead
// editor after route/dialog teardown. Keep this compatibility fix in one place.
export function destroyRichEditor(
  editor: IDomEditor | null | undefined,
  toolbar?: Toolbar | null,
) {
  toolbar?.destroy();
  if (!editor || editor.isDestroyed) return;
  const textarea = DomEditor.getTextarea(editor) as unknown as {
    onDOMSelectionChange?: { cancel?: () => void };
  };
  textarea.onDOMSelectionChange?.cancel?.();
  editor.destroy();
}
