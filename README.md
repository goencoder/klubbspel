# Klubbspel

Ett svenskt hanteringssystem för bordtennisklubbar byggt med Go, React, MongoDB och Protocol Buffers.

## Arkitektur

**Backend**: Go-mikrotjänst med gRPC/REST API:er, MongoDB-databas, protobuf-kodgenerering  
**Frontend**: React TypeScript-applikation med Vite, internationalisering (svenska/engelska)  
**Infrastruktur**: Fly.io-distribution, Docker-containrar, MongoDB Atlas-databas  
**Utveckling**: Protocol buffer-kodgenerering, värdbaserad utveckling för snabb iteration

### Teknikstack

- **Backend**: Go 1.25+, gRPC/REST, MongoDB-drivrutin, protobuf, golangci-lint
- **Frontend**: React 18, TypeScript, Vite, TailwindCSS, i18next
- **Databas**: MongoDB med autentisering och hälsokontroller
- **Distribution**: Fly.io-plattform med Docker flerstegbyggen
- **E-post**: SendGrid (produktion) / MailHog (utveckling)
- **Byggsystem**: Make, buf CLI för protobuf-generering

## Nyckelfunktioner

- ⚽ **Seriehantering** - Skapa tidsbundna turneringsserier med tydliga start-/slutgränser
- 👥 **Spelarregistrering** - Intelligent dubblettidentifiering med normaliserade namn (förhindrar Erik/Eric)
- 🏓 **Matchrapportering** - Rapportera matchresultat med spelpoäng och automatiska ELO-rankinguppdateringar
- 🏆 **Live-resultattavlor** - Offentlig rankningsdisplay i realtid sorterad efter ELO-ranking
- 🌍 **Fullständig internationalisering** - Komplett svenska/engelska språkstöd inklusive felmeddelanden

## 🚀 Snabbstart

### Förkunskaper

- [Go 1.22+](https://golang.org/dl/)
- [Node.js 18+](https://nodejs.org/)
- [Docker & Docker Compose](https://docs.docker.com/get-docker/)
- [Buf CLI](https://docs.buf.build/installation) (för API-utveckling)

### Utvecklingskonfiguration

```bash
# 1. Klona och gå in i katalogen
git clone https://github.com/goencoder/motionsserien.git
cd motionsserien

# 2. Installera beroenden
make host-install

# 3. Starta utvecklingsmiljön
make host-dev
```

### Dagligt utvecklingsarbetsflöde

Den rekommenderade utvecklingsmetoden använder **värdutveckling** för snabb iteration:

1. **Starta utvecklingsmiljön** (MongoDB + MailHog i Docker, backend + frontend på värden):
   ```bash
   make host-dev
   ```

2. **Gör kodändringar** - följ denna ordning för byggen:
   ```bash
   make generate    # Generera protobuf-kod (krävs före backend-bygge)
   make be.build    # Bygg backend
   make fe.build    # Bygg frontend
   ```

3. **Visa applikationen**:
   - **Frontend**: http://localhost:5173 (React dev-server)
   - **Backend API**: http://localhost:8080 (Go-server)
   - **MailHog**: http://localhost:8025 (e-posttestning)

4. **Stoppa utveckling**:
   ```bash
   make host-stop
   ```

### Alternativ: Docker-utveckling

För isolerad utvecklingsmiljö:

```bash
# Starta allt i Docker
make docker-dev

# Stoppa Docker-miljön
make docker-stop
```

## 🛠 Byggkommandon

### Kodgenerering
```bash
make generate        # Generera protobuf-kod (Go + TypeScript)
make proto.clean     # Rensa genererade filer
```

### Backend
```bash
make be.build        # Bygg backend-binär
make be.test         # Kör backend-tester
make be.lint         # Linta backend-kod
```

### Frontend
```bash
make fe.install      # Installera beroenden
make fe.build        # Bygg för produktion
make fe.dev          # Utvecklingsserver
make fe.test         # Kör tester
make fe.lint         # Linta frontend-kod
```

### Fullständig pipeline
```bash
make lint           # Linta både backend och frontend
make build          # Bygg både backend och frontend
make test           # Kör alla tester
```

## 📊 Databashantering

### Utvecklingsdatabas
```bash
make db-up          # Starta MongoDB i Docker
make db-down        # Stoppa MongoDB
make db-reset       # Återställ utvecklingsdata
```

### Testdatabas
```bash
make test-db-up     # Starta testdatabas
make test-db-down   # Stoppa testdatabas
make validate-db    # Kontrollera för inaktuella dataproblem
```

## 📁 Projektstruktur

```
klubbspel/
├── backend/                # Go backend-tjänst
│   ├── cmd/               # Applikationsstartpunkter
│   ├── internal/          # Privat applikationskod
│   │   ├── auth/         # Autentisering & auktorisering
│   │   ├── repo/         # Databasrepositorier
│   │   ├── service/      # Affärslogik
│   │   └── server/       # HTTP/gRPC-servrar
│   ├── proto/gen/go/     # Genererad Go-kod
│   └── openapi/          # OpenAPI-specifikationer
├── frontend/              # React TypeScript frontend
│   ├── src/              # Källkod
│   ├── tests/            # Playwright UI-tester
│   └── dist/             # Byggd frontend (efter bygge)
├── proto/                # Protocol Buffer-definitioner
│   └── pingis/v1/        # API v1-definitioner
├── tests/                # Integrationstester
├── docs/                 # Ytterligare dokumentation
├── bin/                  # Byggda binärer
├── Makefile              # Byggautomation
└── README.md             # Denna fil
```

## 🧰 Teknikstack

### Backend
- **Go 1.22+** - Huvudprogrammeringsspråk
- **gRPC** - Högpresterande RPC-ramverk
- **gRPC-Gateway** - REST API-gateway
- **MongoDB** - Dokumentdatabas
- **Protocol Buffers** - API-definition och serialisering
- **Buf** - Protocol Buffer-verktygskedja

### Frontend
- **React 19** - UI-bibliotek med senaste funktioner
- **TypeScript** - Typsäker JavaScript
- **Vite** - Snabbt byggverktyg och dev-server
- **GitHub Spark** - Komponentramverk
- **Tailwind CSS** - Utility-first CSS-ramverk
- **Playwright** - End-to-end-testning

## 🚀 Distribution

### Distribution till Fly.io

#### Förstagångsinstallation

1. **Installera och autentisera med Fly.io**:
   ```bash
   # Installera flyctl
   curl -L https://fly.io/install.sh | sh
   
   # Logga in på Fly.io
   flyctl auth login
   ```

2. **Konfigurera MongoDB Atlas**:
   - Skapa en MongoDB Atlas-kluster
   - Skapa en databasanvändare
   - Hämta anslutningssträngen (format: `mongodb+srv://username:password@cluster.mongodb.net/pingis`)

3. **Distribuera applikationer**:
   ```bash
   # Distribuera både backend och frontend
   ./deploy.sh full
   ```

4. **Konfigurera hemligheter** (distributionsskriptet visar dig dessa kommandon):
   ```bash
   # Backend-hemligheter
   flyctl secrets set MONGO_URI='mongodb+srv://username:password@cluster.mongodb.net/pingis?retryWrites=true&w=majority' --app klubbspel-backend
   flyctl secrets set MONGO_DB='pingis' --app klubbspel-backend
   
   # E-posthemligheter (om du använder SendGrid)
   flyctl secrets set SENDGRID_API_KEY='your-sendgrid-api-key' --app klubbspel-backend
   flyctl secrets set EMAIL_PROVIDER='sendgrid' --app klubbspel-backend
   
   # Säkerhetshemligheter
   flyctl secrets set GDPR_ENCRYPTION_KEY='your-32-character-encryption-key' --app klubbspel-backend
   ```

#### Iterativa distributioner

För pågående utveckling och distributioner:

```bash
# Distribuera endast backend-ändringar
./deploy.sh backend

# Distribuera endast frontend-ändringar  
./deploy.sh frontend

# Distribuera båda (fullständig distribution)
./deploy.sh full
```

## 🔍 Felsökning

### Loggnivåer

Sätt miljövariabeln `LOG_LEVEL` för att kontrollera loggning:
```bash
export LOG_LEVEL=debug  # debug, info, warn, error
```

### Loggplatser
- **Värdutveckling**: Konsolutput
- **Docker-utveckling**: `docker-compose logs [service]`
- **Produktion**: Konfigurera loggutput via Docker eller systemd

### Vanliga problem

#### MongoDB-anslutningsproblem
```bash
# Kontrollera MongoDB-status
make db-logs

# Återställ databas om den är korrupt
make db-reset
```

#### Frontend-byggproblem
```bash
# Rensa node_modules och installera om
rm -rf frontend/node_modules
make fe.install

# Kontrollera TypeScript-fel
make fe.lint
```

#### Protocol Buffer-problem
```bash
# Regenerera all protobuf-kod
make proto.clean
make generate
```

## 🧪 Testning

### Backend-tester
```bash
make be.test                    # Kör alla backend-tester
go test ./backend/internal/...  # Kör specifika pakettester
```

### Frontend-tester
```bash
make fe.test                    # Kör enhetstester
make fe.test.ui                 # Kör Playwright UI-tester
```

### Integrationstester
```bash
make test                       # Kör fullständig testsvit
make validate-db                # Validera databasintegritet
```

## 📚 Dokumentation

- [Guide för e-postkonfiguration](docs/EMAIL_SETUP.md)
- [Auktoriseringssystem](docs/authz.md)
- [Förebyggande av inaktuell data](docs/STALE_DATA_PREVENTION.md)
- [Sammanfattningar av fasimplementering](PHASE*.md)

## 🤝 Bidra

1. Forka repositoriet
2. Skapa en funktionsgren
3. Följ byggarbetsflödet:
   ```bash
   make lint      # Linta kod
   make generate  # Generera protobuf-kod
   make be.build  # Bygg backend
   make fe.build  # Bygg frontend
   make test      # Kör tester
   ```
4. Skicka in en pull request

## 📄 Licens

Detta projekt är licensierat under MIT-licensen - se [LICENSE](LICENSE)-filen för detaljer.