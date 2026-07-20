# HomeVox Backend

Go API server for HomeVox.

## Run

Build the frontend first, then pass its absolute output directory to the Go server:

```bash
cd frontend
npm ci
npm run build

cd ../backend
HOMEVOX_FRONTEND_DIR="$(cd ../frontend/dist && pwd)" go run ./cmd/server
```

The service uses one fixed public listener, `0.0.0.0:18088`:

- `/api/*` is served by Go.
- Existing frontend assets are served from `HOMEVOX_FRONTEND_DIR`.
- Browser client routes fall back to `index.html`.
- Missing assets and unknown API routes return `404`.

`HOMEVOX_FRONTEND_DIR` is required and must contain `index.html`; startup fails closed when the frontend build is missing. The backend deliberately ignores `HOMEVOX_LISTEN_ADDR` and never drifts from the fixed listener above.
