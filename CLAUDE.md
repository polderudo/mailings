# Mailings — Developer Guide

Newsletter- und Mailing-Anwendung. Basiert auf der vkb-vipsstar Architektur.

## Stack

- **Sprache**: Go 1.26
- **Web Framework**: Echo v4 + JWT Auth
- **Templating**: a-h/templ + templUI + HTMX + Alpine.js + hyperscript
- **UI Components**: nakami-lounge-GmbH/ui-components (git submodule)
- **CSS**: TailwindCSS
- **DB ORM**: bob (stephenafamo/bob, PostgreSQL via pgx)
- **Authorization**: casbin (RbacModelV2) + casbin sql-adapter
- **Mail**: polderudo/mailyak (Transaktionsmails) + Postmark Broadcasts (Newsletter)
- **WYSIWYG-Editor**: TinyMCE 7 Community (self-hosted unter `assets/tinymce/`)
- **i18n**: eigenes typed system (siehe i18n/)
- **Migrationen**: goose (eingebettete SQL)
- **CLI**: cobra
- **Logging**: slog + tint + lumberjack

## Erstes Setup

```bash
# 1. UI-Components submodul holen
git submodule update --init --recursive

# 2. Postgres DB anlegen (siehe scripts/sql/create_db.sql)
psql -U postgres -f scripts/sql/create_db.sql

# 3. Migrationen ausführen
make up

# 4. bob Models gegen die echte DB neu generieren (WICHTIG!)
make bob

# 5. Ersten Admin User anlegen
go run . createUser -s Herr -e admin@example.com -f Admin -l User -p Hawaii11 -a -d

# 6. Dev-Server starten (CSS + templ watch + go run)
task dev
```

## Architektur-Überblick

```
.
├── main.go                  Entry-Point, lädt cmd.Execute()
├── cmd/                     Cobra-Commands: server, migrate, createUser, version, crypt
├── conf/                    Viper-basierte Config (data/config.yaml)
├── core_init/               Bootstrapping (Config → Log → DB → Auth)
├── auth/                    Casbin-Wrapper (Gruppen, Permissions)
├── core/auth/               High-Level Auth (Profile anlegen, Passwort, Token)
│   └── api/                 Admin-API für Benutzerverwaltung
├── core/bob/                System-Queries (Timezone etc.) — handgepflegt
├── srvCtx/                  BobContext mit Transaktions-Helper
├── db/                      DB-Connection, Migrationen, Filter-Utilities
│   └── migrations/          NUR 00001_create_tables.sql (Stub)
├── mq/                      bob-generierte Models (user_profile, token, mail, …)
├── mail/                    SMTP-Versand (mailyak) + DB-Log
├── templates/               text/html-Templates (App-Error, Mail-Body)
├── i18n/                    Typed-i18n-System mit YAML-Source + Code-Generator
├── ilog/                    slog-Setup (Console + Files + Web-Log)
├── internals/               Allgemeine Helper (Crypt, Slice, Time, etc.)
├── mw/                      Echo-Middleware (UserContext, Auth, Lang, WebLog)
├── api/                     HTTP-Routen
│   ├── anon/                Anonyme Routen (Login, Forgot-Pw, Passwort-Setzen)
│   ├── anon/pages/          Server-Side gerenderte Pages (Users, Settings, Home)
│   ├── api_error/           Strukturierte API-Fehler
│   ├── jslogger/            JS-Errors aus dem Browser entgegennehmen
│   └── validate/            Request-Validierung (go-playground/validator)
├── views/                   templ-Views
│   ├── layout/              GuestLayout + UserLayout (Sidebar/Header/Footer)
│   ├── pages/auth/          Login, Forgot-Password, Password-Create
│   ├── pages/main/          Home, Settings, Users (List + Detail)
│   └── pages/errors/        404-Page
├── assets/                  Statische Assets (CSS, JS, Images, Fonts)
├── ui-components/           git submodule mit Form, Input, Button, Table, …
└── data/config.yaml         Haupt-Config (Dev), gemerged mit local.yaml
```

## Unterschiede zu vkb-vipsstar

- **Keine Registrierung**: Benutzer werden ausschließlich über `./app createUser …` angelegt.
  Login-Seite zeigt keinen "Konto erstellen" Link.
- **Nur Migration 00001**: `user_profile`, `user_group`, `token`, `mail`, `client`.
  Die vipsstar-spezifischen Migrationen (CMS, Organization, Branch, Avatar, Sparkasse,
  Page-CMS) sind nicht enthalten.
- **Keine Organization / Branch / Avatar Felder am User-Profil**
- **Kein CMS, kein Sparkasse-Modul, keine Spielregeln / Punktehistorie / Gesamtwertung**
- **Kein MQTT**

## Wichtige Hinweise

### bob Models müssen neu generiert werden
Die `mq/*.bob.go` Dateien wurden für das 00001-Schema händisch angepasst.
Vor produktivem Einsatz unbedingt `make bob` gegen die echte DB ausführen, damit
die bob-Models 1:1 zum Schema passen.

### i18n
Translations liegen in `i18n/data/*.yaml`. Nach Änderung:
```bash
go generate
```
Generiert `i18n/keys.go`, `i18n/vals.go`, `i18n/l.go`. Niemals direkt editieren.

In templ-Komponenten:
```go
i18n.Trl(ctx, i18n.L.Users.AllUsers)
```

In Go-Handlern ohne ctx:
```go
i18n.TrLang(lang, i18n.L.Users.AllUsers)
```

### Routes
Die Anwendung läuft auf Port 3000 (config.yaml). Routes:
- `/auth/login/` — Login-Seite
- `/auth/forgot_password/` — Passwort-Reset anfordern
- `/password_create/:token/` — Passwort über Token setzen
- `/settings/` — Eigene Account-Settings
- `/users/` — Benutzerliste (Admin)
- `/users/:id/` — Benutzerdetail (Admin)
- `/a/...` — JWT-geschützte API-Routen
- `/a/ad/...` — Admin-only API-Routen

### Casbin Policies
Beim Start werden Default-Policies aus `auth/policies.go` geladen. Erweiterungen
dort hinzufügen, neue Operations in `auth/operations.go`.

## Mailing-Modul (Newsletter)

Kern-Feature dieser Anwendung. Workflow:

1. **Template** anlegen / editieren (HTML mit TinyMCE 7 Editor)
2. **Email-Liste** importieren und benennen
3. **Mailing-Vorgang** erstellen (Template + Liste + Domain wählen)
4. **Versand** starten — geht in Postmark Broadcast-Stream

### Tabellen (Migration 00002)

- `mail_template`        — benannte HTML-Templates, im TinyMCE-Editor gepflegt
  - `id`, `name`, `subject`, `body_html`, `created_by`, `created_at`, `updated_at`
- `mail_list`            — benannte Empfänger-Gruppen
  - `id`, `name`, `description`, `created_by`, `created_at`, `updated_at`
- `mail_domain`          — feste Liste der versendbaren Absender-Domains
  - `id`, `domain`, `from_email`, `from_name`, `postmark_stream_id`, `is_active`,
    `created_at`, `updated_at`
- `mailing`              — ein Versand-Vorgang. Empfänger werden NICHT kopiert,
  sondern direkt über `list_id → mail_list_recipient` gelesen. Die Empfänger-
  Anzahl steht **nicht** auf `mailing` (die Liste darf nach Anlegen weiter
  editiert werden); sie wird live über `listsapi.CountListRecipients` gezählt.
  - `id`, `name`, `template_id`, `list_id`, `domain_id`, `status`
    (`draft`, `queued`, `sending`, `done`, `failed`), `subject_snapshot`,
    `body_snapshot`, `started_at`, `finished_at`,
    `sent_count`, `failed_count`, `created_by`, `created_at`, `updated_at`
- `mail_list_recipient`  — einzelne Adressen pro Liste, inkl. letztem Versand-Status
  - Stammdaten: `id`, `list_id` (FK → `mail_list`, ON DELETE CASCADE), `email`,
    `forename`, `lastname`, `created_at`
  - Letzter Versand-Status (überschreibt sich beim nächsten Mailing):
    `last_status` (`""`, `pending`, `sent`, `failed`), `last_postmark_message_id`,
    `last_error`, `last_sent_at`, `last_mailing_id` (FK → `mailing`, ON DELETE SET NULL)
  - UNIQUE (list_id, lower(email))

### TinyMCE Editor (self-hosted)

- TinyMCE 7 Community liegt unter `assets/tinymce/` (GPL 2.0+). Bestandteile:
  `tinymce.min.js`, `models/`, `themes/`, `skins/`, `plugins/`, `icons/`,
  `langs/de.js`. Wird unter `/static/tinymce/...` ausgeliefert.
- Init im Template-Editor-Templ (`tinymceInitScript`): `selector: '#template-editor'`,
  `base_url: '/static/tinymce'`, `suffix: '.min'`, `language: 'de'`,
  `license_key: 'gpl'` (GPL-Akzeptanz). Plugins:
  `lists link image table code autolink searchreplace fullscreen`.
- **Bilder werden inline als `data:image/...;base64,...`** in `body_html`
  gespeichert (keine externen URLs, damit der spätere Versand alles enthält,
  was die Empfänger zum Anzeigen brauchen). Drei Wege landen auf demselben
  Ergebnis:
  1. Drag&Drop / Image-Dialog → `images_upload_handler` macht via FileReader
     eine data-URL daraus.
  2. Paste aus Zwischenablage → `paste_data_images: true`.
  3. „Datei auswählen" im Image-Dialog → `file_picker_callback`.
  Hinweis: einige E-Mail-Clients (z. B. ältere Outlook-Versionen) zeigen
  base64-Bilder nicht zuverlässig; das ist eine bewusste Trade-off-Entscheidung
  zugunsten Autarkie der versendeten Mail.
- Submit-Sync: das Form trägt `hx-on::config-request="window.tmceSyncForm(this)"`
  und schreibt vor jedem htmx-POST `editor.getContent()` in das hidden Input
  `#template-body-html`.
- HTMX-Lifecycle: `htmx:beforeSwap` ruft `tinymce.remove('#template-editor')`,
  `htmx:afterSwap` re-initialisiert den Editor auf dem neu gerenderten Form.
- Lizenzhinweis: `assets/tinymce/license.md` — GPL v2 oder höher.

### HTML-Snippets (`mail/snippet`) — TinyMCE-Toolbar-Menu

Vordefinierte HTML-Bausteine, die im Editor per Toolbar-Menü an die aktuelle
Cursor-Position eingefügt werden.

- Definition in `mail/snippet/snippet.go` als `[]Snippet{ID, Name, HTML}`.
- `core/templates/api` reicht `snippet.All()` über `TemplateFormData.Snippets`
  in die Detail-View.
- View rendert die Liste als JSON in `<script id="template-snippets-data" type="application/json">`.
- TinyMCE-Setup registriert via `editor.ui.registry.addMenuButton('snippets', { … })`
  ein Dropdown in der Toolbar; jeder Klick auf einen Eintrag ruft
  `editor.insertContent(s.html)` — das ist Quill-frei und ohne die früheren
  Selection-/DOM-Crashes.

### Postmark Broadcasts

- Wir nutzen **Postmark Broadcasts** (kein eigenes SMTP):
  - Doku: https://postmarkapp.com/developer/api/broadcasts-api
- Konfiguration in `data/config.yaml` → `postmark.server_token` + Domain-spezifischer Stream
- Modul `mail/postmark/` (nach ausbauen) kapselt den Versand
- Bei `POST /a/mailings/:id/start/` wird der Job asynchron in `mq/mailings_worker.go`
  abgearbeitet (oder synchron als MVP, je nach Stand)

### Module (Mailing-Modul)

Alle Handler und Routen liegen unter `core/` und sind **vollständig** über
`rUser` (JWT-Authentifizierung) registriert. Pages und API teilen sich denselben
`/a/...`-Prefix.

| Pfad | Methode | Zweck |
|------|---------|-------|
| `/a/templates/` | GET | Liste aller Templates |
| `/a/templates/new/` | GET | Neues Template anlegen (TinyMCE) |
| `/a/templates/:id/` | GET | Editor für vorhandenes Template |
| `/a/templates/` | POST | Template anlegen |
| `/a/templates/:id/` | POST | Template aktualisieren |
| `/a/templates/:id/` | DELETE | Template löschen |
| `/a/lists/` | GET | Liste aller Email-Listen |
| `/a/lists/new/` | GET | Neue Email-Liste anlegen + Import |
| `/a/lists/:id/` | GET | Detailansicht Email-Liste |
| `/a/lists/` | POST | Liste anlegen |
| `/a/lists/:id/` | POST | Liste aktualisieren |
| `/a/lists/:id/import/` | POST | Empfänger importieren (CSV/Textarea) |
| `/a/lists/:id/` | DELETE | Liste löschen |
| `/a/mailings/` | GET | Liste aller Mailing-Vorgänge |
| `/a/mailings/new/` | GET | Mailing anlegen (Template/Liste/Domain) |
| `/a/mailings/:id/` | GET | Detailansicht eines Mailings |
| `/a/mailings/` | POST | Mailing anlegen |
| `/a/mailings/:id/start/` | POST | Versand starten |

### Code-Layout (Mailing-Modul)

- `core/templates/api/`  — Handler + `CreateRoutes(rUser)`
- `core/lists/api/`      — Handler + `CreateRoutes(rUser)` + `LoadListRecipients`/`CountListRecipients`
- `core/mailings/api/`   — Handler + `CreateRoutes(rUser)`
- `db/templates_table.go`, `db/lists_table.go`, `db/mailings_table.go` — Tabellen-Loader
- `mail/postmark/`       — Postmark-Broadcast-Client (Stub, TODO HTTP-Call)
- `views/pages/main/templates/`, `views/pages/main/lists/`, `views/pages/main/mailings/` — templ-Views
- Registriert in `api/routes.go` via:
  ```go
  templatesApi.CreateRoutes(rUser)
  listsApi.CreateRoutes(rUser)
  mailingsApi.CreateRoutes(rUser)
  ```

## Named Routes — `api/router` (verbindlich, projektweit)

Alle Routen — anon, authentifiziert, admin, JsLog, Auth, Settings, Users,
Templates, Lists, Mailings — werden ausschließlich als **named routes**
registriert und in den Views/Handlern über `router.Reverse(...)` aufgelöst.
Hardcodierte URL-Strings (`/a/templates/`, `"/auth/login/"`,
`fmt.Sprintf("/a/...", id)`) sind nicht erlaubt — auch nicht für Form-Actions,
`hx-*`-Attribute, `<a href>`, `HX-Redirect`-Header oder `c.Redirect`.

### Aufbau

- `api/router/main.go`        — `ERouter` (Echo-Instanz), `router.Reverse(name, params...)`, `EnsureAllFieldsSet`.
- `api/router/route_names.go` — typed structs pro Modul mit den Routennamen.
  Jede Group wird in einem `init()` per `EnsureAllFieldsSet` validiert,
  damit Tippfehler beim Start auffallen.
- `mw.UserGroup.AddNamedRoute(name, path, handler, methods...)` — registriert
  einen benannten Endpunkt für JWT-geschützte Routen. Erste Methode trägt den `Route.Name`.
- `mw.AnonGroup.AddNamedRoute(name, path, handler, methods...)` — analog für
  öffentliche Routen mit `echo.HandlerFunc`.
- `mw.AnonGroup.AddNamedAnonRoute(name, path, handler, methods...)` — analog für
  Handler mit `func(*mw.AnonUserContext) error`-Signatur (Login-Page, Index, …).

### Regeln

1. **Neue Route → Eintrag in `api/router/route_names.go`** (passende Group
   oder neue Group anlegen) und `EnsureAllFieldsSet(<Group>)` in einem `init()`.
2. **Im Handler-Modul** Routen ausschließlich via
   `rUser.AddNamedRoute(router.<Group>.<Name>, "/path/", handler, http.MethodGet, ...)`
   registrieren — niemals `rUser.GET(...)` oder `EchoGroup.POST(...)`.
3. **In templ-Views** URLs immer mit `router.Reverse(router.<Group>.<Name>, params...)`
   bilden. Pfadparameter werden in der Reihenfolge der `:param`-Tokens übergeben.
   Beispiel: `router.Reverse(router.Templates.Detail, strconv.Itoa(int(id)))`.
4. **Tabellen-RowClickURLPattern**: `router.Reverse(router.<Group>.Detail, "{value}")` —
   Echo setzt den `{value}`-Token in den `:id`-Slot ein, die ui-component ersetzt
   ihn auf der Client-Seite.
5. **Server-seitige Redirects** (`HX-Redirect`, `c.Redirect`) immer via
   `router.Reverse`, nicht via String-Templates auf Konstanten.
6. **Keine Routen-Konstanten** mehr in `views/.../*.go` (`*PageRoute`, `*APIRoute`,
   `fmt.Sprintf(..., id)`). Die einzige Quelle der Wahrheit ist
   `api/router/route_names.go` + die `path` aus `AddNamedRoute`.
7. **ERouter setzen**: `api/main.go` weist direkt nach `echo.New()` der Variable
   `router.ERouter` die Echo-Instanz zu, sonst gibt `Reverse` einen leeren String.

### Sidebar
Eintrag "Mailings" mit Untermenüpunkten Templates, Listen, Domains, Mailings.
Liegt in `views/layout/components/sidebar.templ`.

## Excel-Import (`nakami-lounge-GmbH/tools/importer`)

Für Datei-Uploads, die in der Anwendung als Excel verarbeitet werden, nutzen wir
`importer.NewExcelLineImporter` aus dem Paket
`github.com/nakami-lounge-GmbH/tools/importer`. Das aktuelle Beispiel ist der
Empfänger-Import in `core/lists/api/lists.go`.

### Schema

Pro Import-Format eine Row-Struktur mit `header:"..."`-Tags, die exakt den
Spaltenüberschriften in der Excel-Datei entsprechen (case-insensitiv,
getrimmt). Optional Validator-Tags (`validate:"required,email"` etc.).

```go
type excelRecipientRow struct {
    Email    string `header:"Email" validate:"required,email"`
    Forename string `header:"Forename"`
    Lastname string `header:"Lastname"`
}
```

### Aufruf

```go
fh, _ := c.FormFile("file")
f, _ := fh.Open(); defer f.Close()
bytes, _ := io.ReadAll(f)

eL := &importer.ErrorList{}
imp, err := importer.NewExcelLineImporter[excelRecipientRow](&importer.ExcelLineConfig{
    Config: importer.Config{
        SheetNumber: 1,   // 1-basiert; alternativ SheetName
        OffsetRow:   1,   // Zeile mit Headern (1-basiert), Daten ab darauf folgender Zeile
        FileBytes:   bytes,
    },
}, eL)
if err != nil || eL.HasErrors() { ... }

for _, row := range imp.Data { ... }
```

### Upload-Komponente

Für den Frontend-Upload nutzen wir die wiederverwendbare Dropzone aus
`ui-components/dropzone`:

```templ
@uidropzone.Dropzone(uidropzone.Props{
    Name:              "file",
    AllowedExtensions: []string{"xlsx"},
    MaxSizeBytes:      10 * 1024 * 1024,
    Description:       Trl(ctx, L.Lists.ExcelHeaderColumnsEmailForenameLastname),
    Required:          true,
})
```

Die Dropzone rendert keine eigene `<form>` — sie wird in eine umschließende
Form mit `enctype="multipart/form-data"` (und `hx-encoding="multipart/form-data"`
falls HTMX-getrieben) eingesetzt. Drag-and-Drop sowie clientseitige
Format-/Größenprüfung sind über Alpine integriert; eigene Fehlertexte können
per Props oder über `dropzone.LangMap` lokalisiert werden.

> `ui-components/avatar/dropdzone.templ` ist die ältere, avatar-spezifische
> Variante mit Bild-Preview. Sie soll perspektivisch durch die neue
> `dropzone`-Komponente abgelöst werden.

## gograph MCP Server

In `.mcp.json` ist ein `gograph` MCP-Server registriert. Workflow:

1. Bei Session-Start: `gograph_capabilities` aufrufen.
2. NIE `grep/rg/find/glob` zum Suchen von Go-Symbolen verwenden — stattdessen
   `gograph_query`.
3. Vor Edits an Symbolen: `gograph_plan` mit `with_context=true`.
4. Nach Go-Edits: `gograph_review` mit `uncommitted=true`.
5. Symbol verstehen: `gograph_context` (uncommitted=true für Sammel-Lookup).

## Hilfsbefehle

```bash
# DB-Migration ausführen
make up
# down auf eine Version
GOOSE_TO_VERSION=1 make down
# Bob-Modelle neu generieren (nach Migration)
make bob
# i18n-Code regenerieren (nach YAML-Änderung)
go generate ./i18n/...
# Build
make api
# Dev-Server
task dev
```

