# Local development

```bash
make setup    # .env, Postgres/Redis, pnpm install
make migrate
make dev      # backend :8080 (hot reload) + frontend :3000 (HMR)
```

## Backend hot reload

`make dev` uses [Air](https://github.com/air-verse/air): при сохранении `.go` сервер пересобирается и перезапускается.

```bash
env -u GOROOT go install github.com/air-verse/air@latest
export PATH="$(env -u GOROOT go env GOPATH)/bin:$PATH"   # если air not found
make backend-watch   # только API
```

Без Air: `make backend` — один запуск через `go run`.

### Ошибка `go1.26.0 does not match go tool version go1.26.3`

Часто из‑за `GOROOT` от goenv на старой версии при бинарнике Go 1.26.3. Варианты:

```bash
env -u GOROOT go install github.com/air-verse/air@latest   # разово
goenv install 1.26.3 && goenv local 1.26.3                  # в каталоге проекта
```

`make backend-watch` уже запускает `air` с `env -u GOROOT`.

All targets: `make help`

## Checks

```bash
make test
make typecheck
make smoke      # server must be on :8080
make coverage   # CI gate
```
