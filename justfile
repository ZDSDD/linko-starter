set shell := ["bash", "-euo", "pipefail", "-c"]

port := env_var_or_default("LINKO_PORT", "8899")
data_dir := env_var_or_default("LINKO_DATA_DIR", "./data")
log_file := env_var_or_default("LINKO_LOG_FILE", "linko.access.log")
environment := env_var_or_default("ENV", "development")
binary := env_var_or_default("LINKO_BINARY", "./linko")
app_url := "http://localhost:" + port

# Pokaż dostępne komendy
default:
    @just --list

# Uruchom aplikację przez go run
run:
    LINKO_LOG_FILE="{{ log_file }}" ENV="{{ environment }}" go run . -port "{{ port }}" -data "{{ data_dir }}"

# Zbuduj aplikację z Git SHA i czasem buildu
build:
    go build -ldflags "-X boot.dev/linko/internal/build.GitSHA=$(git rev-parse HEAD) -X boot.dev/linko/internal/build.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" -o "{{ binary }}" .

# Zbuduj i uruchom binarkę
start: build
    LINKO_LOG_FILE="{{ log_file }}" ENV="{{ environment }}" "{{ binary }}" -port "{{ port }}" -data "{{ data_dir }}"

# Uruchom monitoring w tle, a aplikację na pierwszym planie
dev: stack-bg run

# Uruchom testy Go
test:
    go test -v ./...

# Uruchom analizę statyczną Go
vet:
    go vet ./...

# Uruchom testy i analizę statyczną
check: test vet

# Sformatuj kod Go
fmt:
    go fmt ./...

# Uruchom test lekcji; opcjonalnie podaj UUID
lesson lesson="":
    bootdev run {{ lesson }}

# Sprawdź i wyślij lekcję; opcjonalnie podaj UUID
submit lesson="":
    bootdev run {{ lesson }} -s

_docker-ready:
    @docker compose version >/dev/null 2>&1 || { printf '%s\n' 'Docker Compose nie działa. W WSL włącz integrację tej dystrybucji w Docker Desktop.' >&2; exit 1; }

# Uruchom Prometheusa, Grafanę i node-exporter na pierwszym planie
stack: _docker-ready
    docker compose up

# Uruchom monitoring w tle
stack-bg: _docker-ready
    docker compose up -d

# Zatrzymaj monitoring
stack-down: _docker-ready
    docker compose down

# Śledź logi monitoringu
stack-logs: _docker-ready
    docker compose logs -f

# Pokaż stan kontenerów
stack-status: _docker-ready
    docker compose ps

# Pobierz surowe metryki Linko
metrics:
    curl -fsS "{{ app_url }}/metrics"

# Wygeneruj ruch; domyślnie 3500 żądań
traffic count="3500":
    LINKO_URL="{{ app_url }}" REQUEST_COUNT="{{ count }}" ./spamhomepage.sh

# Pokaż przydatne adresy
urls:
    @printf '%s\n' \
        'Linko:      {{ app_url }}' \
        'Metrics:    {{ app_url }}/metrics' \
        'Prometheus: http://localhost:9090' \
        'Grafana:    http://localhost:3000 (admin/admin)' \
        'Node:       http://localhost:9100/metrics'
