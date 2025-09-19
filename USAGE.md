# 📖 GoIOTop - Guia Completo de Uso

## 🚀 Início Rápido

### Instalação

```bash
# Clone o repositório
git clone https://github.com/reinaldosaraiva/goiotop.git
cd goiotop

# Instale as dependências
go mod download

# Compile o projeto
go build -o goiotop cmd/goiotop/*.go

# Ou use o Makefile
make build
```

## 💻 Modos de Uso

### 1. TUI (Interface Terminal Interativa)

O modo mais comum e visual para monitoramento em tempo real.

```bash
# Iniciar interface interativa
./goiotop tui

# Com opções personalizadas
./goiotop tui --refresh 2s --theme dark --process-limit 50

# Modo verbose para debug
./goiotop tui --verbose
```

#### Controles do TUI:

| Tecla | Ação |
|-------|------|
| `↑/↓` | Navegar pela lista de processos |
| `Tab` | Alternar entre abas |
| `s` | Alternar ordenação (CPU/Memory/IO) |
| `f` | Filtrar processos |
| `k` | Kill processo selecionado |
| `p` | Pausar/Continuar atualização |
| `g` | Ir para o topo da lista |
| `G` | Ir para o fim da lista |
| `h` | Mostrar/ocultar ajuda |
| `q` ou `Ctrl+C` | Sair |

#### Temas Disponíveis:
- `default` - Tema padrão com cores balanceadas
- `dark` - Tema escuro para ambientes com pouca luz
- `light` - Tema claro para terminais claros
- `monochrome` - Sem cores, apenas texto

### 2. Gráficos Históricos

Visualize métricas históricas em gráficos ASCII.

```bash
# Gráfico de CPU da última hora
./goiotop graphs --metric cpu --duration 1h

# Gráfico de memória das últimas 24 horas
./goiotop graphs --metric memory --duration 24h

# Gráfico de IO de disco dos últimos 15 minutos
./goiotop graphs --metric disk --duration 15m

# Gráfico customizado com dimensões
./goiotop graphs --metric cpu --duration 1h --width 120 --height 30

# Gráfico de processo específico
./goiotop graphs --metric process --pid 1234 --duration 30m
```

#### Métricas Disponíveis:
- `cpu` - Uso de CPU do sistema
- `memory` - Uso de memória RAM
- `disk` - I/O de disco (leitura/escrita)
- `process` - Métricas de processo específico
- `network` - I/O de rede (futura implementação)

### 3. Sistema de Alertas

Configure e monitore alertas baseados em thresholds.

```bash
# Listar alertas ativos
./goiotop alerts list

# Ver histórico de alertas
./goiotop alerts history

# Adicionar regra de alerta
./goiotop alerts add --metric cpu --threshold 80 --duration 5m --severity high

# Remover regra de alerta
./goiotop alerts remove --id alert-123

# Testar configuração de alertas
./goiotop alerts test
```

#### Configuração de Alertas (config.yaml):

```yaml
alerting:
  enabled: true
  rules:
    - name: "High CPU Usage"
      metric: "cpu"
      threshold: 80.0
      operator: ">"
      duration: "5m"
      severity: "warning"

    - name: "Memory Critical"
      metric: "memory"
      threshold: 90.0
      operator: ">"
      duration: "2m"
      severity: "critical"

    - name: "Disk IO Saturation"
      metric: "disk_util"
      threshold: 95.0
      operator: ">"
      duration: "1m"
      severity: "warning"
```

### 4. Exportador Prometheus

Exponha métricas para coleta do Prometheus.

```bash
# Iniciar exportador na porta padrão (9090)
./goiotop export

# Porta customizada
./goiotop export --prometheus-port 8080

# Com path customizado
./goiotop export --prometheus-path /custom/metrics

# Executar em background
nohup ./goiotop export > goiotop.log 2>&1 &
```

#### Configurar Prometheus para coletar:

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'goiotop'
    static_configs:
      - targets: ['localhost:9090']
    scrape_interval: 15s
```

#### Queries Prometheus Úteis:

```promql
# CPU usage médio nos últimos 5 minutos
avg_over_time(goiotop_cpu_usage_percent[5m])

# Top 10 processos por uso de memória
topk(10, goiotop_process_memory_rss_bytes)

# Taxa de I/O de disco
rate(goiotop_disk_read_bytes_total[1m])

# Alertas para CPU alta
goiotop_cpu_usage_percent > 80
```

## ⚙️ Configuração

### Arquivo de Configuração (config.yaml)

```yaml
# config.yaml
collection:
  interval: 1s
  timeout: 5s
  batch_size: 100
  enable_cpu: true
  enable_memory: true
  enable_disk: true
  enable_process: true
  process_filters:
    include_idle: false
    include_kernel: false
    min_cpu_percent: 0.1
    min_memory_mb: 10

display:
  refresh_rate: 1s
  max_rows: 50
  show_system_stats: true
  show_process_list: true
  show_graphs: true
  color_scheme: "dark"

storage:
  type: "memory"
  retention_period: 24h
  max_metrics: 100000

export:
  prometheus:
    enabled: true
    port: 9090
    path: "/metrics"
    include_processes: true
    process_limit: 100

logging:
  level: "info"
  format: "json"
  output: "stdout"
```

### Usar com arquivo de configuração:

```bash
# Especificar arquivo de config
./goiotop tui --config /path/to/config.yaml

# Config padrão em ~/.goiotop/config.yaml
./goiotop tui

# Override de configurações via flags
./goiotop tui --config config.yaml --refresh 2s
```

## 📊 Casos de Uso Comuns

### Monitoramento de Servidor Web

```bash
# Monitorar apenas processos web
./goiotop tui --filter "nginx|apache|httpd"

# Exportar métricas para Prometheus
./goiotop export --process-filter "nginx"
```

### Debug de Performance

```bash
# Alta frequência de atualização
./goiotop tui --refresh 500ms --verbose

# Focar em processos com alto I/O
./goiotop tui --sort io --only-active
```

### Monitoramento de Container

```bash
# Dentro de container Docker
docker run --rm -it --pid host --network host \
  -v /proc:/host/proc:ro \
  goiotop:latest tui

# Em Kubernetes
kubectl exec -it pod-name -- goiotop tui
```

### Automação e Scripts

```bash
# Coletar métricas em JSON
./goiotop collect --format json --output metrics.json

# Pipe para processamento
./goiotop collect --format csv | awk -F',' '{print $2,$3}'

# Monitoramento contínuo com log
./goiotop tui 2>&1 | tee -a monitoring.log
```

## 🔧 Solução de Problemas

### Problema: "Permission denied" ao ler /proc

```bash
# Executar com sudo (não recomendado em produção)
sudo ./goiotop tui

# Ou adicionar ao grupo de monitoramento
sudo usermod -a -G monitoring $USER
```

### Problema: TUI não renderiza corretamente

```bash
# Verificar suporte do terminal
echo $TERM

# Forçar modo sem cores
./goiotop tui --no-color

# Usar tema monochrome
./goiotop tui --theme monochrome
```

### Problema: Alto uso de CPU pelo GoIOTop

```bash
# Reduzir frequência de coleta
./goiotop tui --refresh 5s

# Limitar número de processos
./goiotop tui --process-limit 50

# Desabilitar features não necessárias
./goiotop tui --no-graphs --no-system-stats
```

### Problema: Métricas incorretas ou faltando

```bash
# Modo debug para ver erros
./goiotop tui --verbose --log-level debug

# Verificar permissões
ls -la /proc/stat /proc/meminfo

# Testar collectors individualmente
./goiotop test --collector system
./goiotop test --collector process
```

## 🎯 Exemplos Avançados

### Pipeline de Monitoramento Completo

```bash
#!/bin/bash
# monitoring-pipeline.sh

# 1. Iniciar exportador em background
./goiotop export --prometheus-port 9090 &
EXPORTER_PID=$!

# 2. Iniciar coleta de alertas
./goiotop alerts monitor --output alerts.log &
ALERT_PID=$!

# 3. TUI interativo para visualização
./goiotop tui --config production.yaml

# Cleanup
kill $EXPORTER_PID $ALERT_PID
```

### Integração com Grafana

```json
{
  "dashboard": {
    "title": "GoIOTop Metrics",
    "panels": [
      {
        "title": "CPU Usage",
        "targets": [
          {
            "expr": "goiotop_cpu_usage_percent",
            "legendFormat": "CPU %"
          }
        ]
      },
      {
        "title": "Top Processes by Memory",
        "targets": [
          {
            "expr": "topk(5, goiotop_process_memory_rss_bytes)",
            "legendFormat": "{{name}} ({{pid}})"
          }
        ]
      }
    ]
  }
}
```

### Script de Análise Automatizada

```python
#!/usr/bin/env python3
# analyze_metrics.py

import subprocess
import json
import pandas as pd

# Coletar métricas
result = subprocess.run(
    ['./goiotop', 'collect', '--format', 'json'],
    capture_output=True, text=True
)

# Processar dados
data = json.loads(result.stdout)
df = pd.DataFrame(data['processes'])

# Análise
print(f"Total processos: {len(df)}")
print(f"CPU médio: {df['cpu_percent'].mean():.2f}%")
print(f"Top 5 por memória:")
print(df.nlargest(5, 'memory_rss')[['name', 'pid', 'memory_rss']])

# Detectar anomalias
high_cpu = df[df['cpu_percent'] > 80]
if not high_cpu.empty:
    print(f"⚠️ Processos com CPU alta: {high_cpu['name'].tolist()}")
```

## 📱 Integração com Ferramentas

### Slack Notifications

```bash
# alert-to-slack.sh
#!/bin/bash

WEBHOOK_URL="https://hooks.slack.com/services/YOUR/WEBHOOK/URL"

./goiotop alerts list --format json | jq -r '.active[] |
  "🚨 Alert: \(.name) - \(.severity) - \(.message)"' |
  while read alert; do
    curl -X POST -H 'Content-type: application/json' \
      --data "{\"text\":\"$alert\"}" $WEBHOOK_URL
  done
```

### Datadog Integration

```yaml
# datadog-agent.yaml
logs:
  - type: file
    path: /var/log/goiotop/*.log
    service: goiotop
    source: go

metrics:
  - type: prometheus
    url: http://localhost:9090/metrics
    namespace: goiotop
    metrics:
      - goiotop_*
```

## 📚 Referência Rápida

### Flags Globais

| Flag | Descrição | Padrão |
|------|-----------|--------|
| `--config, -c` | Arquivo de configuração | `~/.goiotop/config.yaml` |
| `--verbose, -v` | Modo verbose | `false` |
| `--log-level` | Nível de log | `info` |
| `--help, -h` | Mostrar ajuda | - |

### Variáveis de Ambiente

```bash
export GOIOTOP_CONFIG="/path/to/config.yaml"
export GOIOTOP_LOG_LEVEL="debug"
export GOIOTOP_PROMETHEUS_PORT="9090"
export GOIOTOP_THEME="dark"
```

### Atalhos Úteis

```bash
# Alias para comandos comuns
alias iotop='goiotop tui'
alias iotop-graphs='goiotop graphs --metric cpu --duration 1h'
alias iotop-export='goiotop export --prometheus-port 9090'

# Função para monitoramento rápido
monitor() {
  goiotop tui --filter "$1" --refresh 1s
}

# Uso: monitor nginx
```

## 🆘 Suporte

- **Issues**: [GitHub Issues](https://github.com/reinaldosaraiva/goiotop/issues)
- **Documentação**: [README.md](README.md)
- **Exemplos**: [examples/](examples/)

---

**Versão**: 1.0.0 | **Última atualização**: Janeiro 2024