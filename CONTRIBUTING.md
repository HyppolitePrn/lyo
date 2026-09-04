# Contributing to Lyo

*Version française en fin de document · French version at the end.*

## Branching and commits

**Branch naming:** `type/short-description`, mirroring the commit type — `feat/broadcaster-screen`,
`fix/jwt-expiry`, `ci/lint-step`, `docs/user-guide`.

**Commit format:** `type(scope): message`

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`.

**Signing.** Commits should be signed (`git config commit.gpgsign true`) so they show as *Verified* on
GitHub. Signed commits are part of the project's supply-chain integrity story.

## Before opening a pull request

These are the same checks CI runs. Run them locally — a red pipeline costs everyone more than two
minutes of waiting.

```bash
# Backend
cd backend && golangci-lint run ./... && go build ./... && go test -race ./...

# Mobile
cd mobile && flutter analyze && flutter test
```

## Quality gates (enforced by CI, not by convention)

| Gate | Rule |
|---|---|
| Static analysis | `golangci-lint` v2 and `flutter analyze` must be clean |
| Race detector | `go test -race` is always on |
| **Coverage** | **≥ 80 %**, excluding generated code — a PR that drops below this **fails** |
| Contract integrity | `go generate ./internal/api/` then compile: contract/code drift fails the build |
| Build | `go build ./cmd/server` must succeed |

## Rules that are specific to this codebase

1. **Never edit `internal/api/api.gen.go`.** It is generated from `backend/api/openapi.yaml`. Change the
   contract, then run `go generate ./internal/api/...`.
2. **Every handler derives a timeout context** at the top:
   ```go
   ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
   defer cancel()
   ```
   Return **503** (not 500) on `context.DeadlineExceeded`. Reference timeouts: simple query 5 s, complex
   query 10 s, external HTTP call 10 s, audio stream init 15 s.
3. **Every new feature sits behind a feature flag.** Add the flag to `internal/features/seed.go` *before*
   implementing the feature, and gate the handler at the top.
4. **Authorization lives in the handler, not the middleware.** `Authenticate` is permissive by design
   (see [ADR 007](docs/adr/007-permissive-auth-middleware.md)): it enriches the context and never
   rejects. Your handler must check claims, role (`AtLeast`, never equality), feature flag, and ownership
   — and a test must assert each refusal path.
5. **No hardcoded configuration.** Everything goes through `pkg/config`, sourced from environment
   variables, and is documented in `.env.example`.
6. **Tests ship in the same pull request as the code they cover.** A bug fix ships with a regression test
   that fails before the fix and passes after it.
7. **Structural decisions get an ADR** in `docs/adr/`, written when the decision is made. If a decision
   is reversed, mark the old record *Superseded* and write a new one explaining why — do not delete or
   quietly edit it.
8. **Documentation states reality.** Use ✅ for what is delivered and 🚧 for what is planned. Never
   describe an intention as an achievement.
9. **Accessibility is a functional requirement.** Icon-only controls need accessible labels, touch
   targets are ≥ 48 × 48 dp, and no information is carried by colour alone.

## Reviewing a pull request

- Are the **error paths** tested, not just the happy path?
- Are role, feature flag and ownership all checked where they apply?
- Does the change need a documentation or ADR update?
- Does anything in the diff contradict a document that was not updated?

---

## Contribuer (français)

**Branches :** `type/description-courte`, en miroir du type de commit.
**Commits :** `type(scope): message` — types : `feat`, `fix`, `refactor`, `test`, `docs`, `chore`, `ci`.
Les commits doivent être signés.

**Avant toute pull request**, exécuter localement les mêmes contrôles que la CI :
`golangci-lint run ./... && go build ./... && go test -race ./...` côté backend,
`flutter analyze && flutter test` côté mobile.

**Contrôles bloquants en CI :** analyse statique propre, détecteur de compétition activé, **couverture
≥ 80 %** hors code généré, régénération du contrat OpenAPI sans dérive, build réussi.

**Règles propres au projet :** ne jamais modifier `api.gen.go` ; dériver un contexte à échéance en tête
de chaque handler et renvoyer 503 sur dépassement ; placer toute nouvelle fonctionnalité derrière un
interrupteur ; contrôler rôle, interrupteur et propriété **dans le handler** (le middleware est
permissif par conception) et tester chaque chemin de refus ; aucune valeur de configuration en dur ;
tests livrés dans la même PR que le code ; ADR pour toute décision structurante, marqué *Superseded* en
cas de revirement ; documentation qui distingue explicitement le livré (✅) du planifié (🚧) ;
accessibilité traitée comme une exigence fonctionnelle.
