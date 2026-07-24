# Issue #19 design–implementation parity

Frozen design evidence: Issue #19 comment `5068143912`, 1440 × 960 representative states.
This implementation intentionally uses a controlled fixture only for automated evidence; no user floorplan is stored in this repository.

| Frozen state | Implementation evidence | Deliberate difference and reason |
| --- | --- | --- |
| 导入 / AI 识别 | Step 1/2 sidebar, browser file picker, image facts, `开始 AI 识别`, loading/error/retry backed by `POST /api/floorplans/parse` | Uses native file input for keyboard and screen-reader support. |
| 校正可编辑 2D | Step 3, original-image overlay, stable wall/opening controls, shared Undo/Redo, Inspector | “补画墙体”和“对象确认” are absent: no verified canonical contract exists. |
| 生成可编辑 3D | Step 4, real R3F/WASM panel and same-document statement | Wall height remains an explicit preview; geometry engine telemetry is test-only, not ordinary UI. |
| 2D / 3D 联动 | Step 5, equal workspace panels, shared selection and one Undo/Redo history | The responsive implementation stacks panels below 1024px for usable zoom and focus order. |
| 保存项目 | Step 6 and Inspector project card use verified create/update/load endpoints | Manual save is shown instead of automatic save because autosave/conflict semantics have no accepted contract. |

## Exact-head evidence command

Run `npm --prefix frontend run test:e2e` on the PR head. The production Playwright
suite uses a fixed 1440 × 960 viewport and captures `issue-19-import-ai.png`,
`issue-19-2d-correction.png`, `issue-19-3d-confirm.png`, and
`issue-19-linked-workspace.png` as test artifacts. It also verifies keyboard-visible
native controls, ARIA labels, retry, invalid geometry, WebGL/WASM fallback, and the
same canonical project across save/reload.
