# Инструкции по интеграции и использованию компонента ClickHouse Exporter для Grafana Alloy

## 1. Обзор

Этот документ описывает, как интегрировать и использовать компонент `otelcol.exporter.clickhouse` в Grafana Alloy. Данный компонент является "community component" и предназначен для экспорта данных OpenTelemetry (логи, трассировки, метрики) в ClickHouse.

**Экспериментальный статус:**

Компонент `otelcol.exporter.clickhouse` является экспериментальным. Это означает,_что он не покрывается стандартными гарантиями обратной совместимости Grafana Alloy и его API может измениться в будущих версиях. Используйте его с осторожностью в производственных средах.

## 2. Активация Community Components

Для использования `otelcol.exporter.clickhouse` и других community components, необходимо запустить Grafana Alloy с флагом `--feature.community-components.enabled=true`.

Пример запуска Grafana Alloy:

```bash
./alloy-linux-amd64 run --feature.community-components.enabled=true config.alloy
```

## 3. Сборка Grafana Alloy с компонентом (если требуется)

На данный момент, для использования этого community компонента, предполагается, что он будет включен в один из будущих релизов Grafana Alloy или вы будете использовать кастомную сборку Alloy, в которую данный компонент интегрирован.

Если вы разрабатываете или тестируете компонент локально, вам потребуется добавить исходный код компонента в дерево исходных кодов Grafana Alloy (обычно в директорию типа `internal/component/otelcol/exporter/clickhouseexporter/`) и пересобрать Alloy согласно инструкциям для разработчиков Grafana Alloy.

Общие шаги для сборки Alloy из исходников (могут отличаться в зависимости от версии и окружения):

1.  Клонировать репозиторий Grafana Alloy: `git clone https://github.com/grafana/alloy.git`
2.  Перейти в директорию с исходным кодом Alloy: `cd alloy`
3.  Разместить код компонента `clickhouseexporter` в соответствующей директории (например, `internal/component/otelcol/exporter/clickhouseexporter/`).
4.  Убедиться, что компонент зарегистрирован в `init()` функции пакета.
5.  Собрать Alloy: `go build ./cmd/alloy` (или используя `make` если доступно).

## 4. Конфигурация компонента

Компонент `otelcol.exporter.clickhouse` настраивается в конфигурационном файле Grafana Alloy (обычно `config.alloy`) в формате River.

### Пример конфигурационного файла (`config.alloy`)

```river
// Включение удаленной конфигурации (если используется)
// remote.http "config_source" {
//   url = "http://localhost:8080/config.alloy"
//   poll_interval = "1m"
// }

// Пример простого конвейера: OTLP Receiver -> Batch Processor -> ClickHouse Exporter

otelcol.receiver.otlp "default" {
  grpc {
    endpoint = "0.0.0.0:4317"
  }
  http {
    endpoint = "0.0.0.0:4318"
  }
  output {
    logs   = [otelcol.processor.batch.default.input]
    traces = [otelcol.processor.batch.default.input]
    metrics= [otelcol.processor.batch.default.input]
  }
}

otelcol.processor.batch "default" {
  output {
    logs   = [otelcol.exporter.clickhouse.default.input]
    traces = [otelcol.exporter.clickhouse.default.input]
    metrics= [otelcol.exporter.clickhouse.default.input]
  }
}

otelcol.exporter.clickhouse "default" {
  dsn = "tcp://localhost:9000/default?read_timeout=20s&write_timeout=20s&dial_timeout=5s"
  timeout = "10s"

  logs {
    table    = "my_alloy_logs"
    ttl_days = 7
  }

  traces {
    table    = "my_alloy_traces"
    ttl_days = 7
  }

  metrics {
    table    = "my_alloy_metrics" // Базовое имя, суффиксы _gauge, _sum и т.д. будут добавлены автоматически
    ttl_days = 30
  }

  retry_on_failure {
    enabled = true
    initial_interval = "10s"
    max_interval     = "1m"
    max_elapsed_time = "10m"
  }

  sending_queue {
    enabled = true
    num_consumers = 5
    queue_size    = 500
  }

  clickhouse {
    // cluster_name = "my_cluster" // Если используется кластер ClickHouse
    // table_engine = "ReplicatedMergeTree('/clickhouse/tables/{shard}/{database}/{table}', '{replica}')" // Пример для ReplicatedMergeTree
  }
}

// Можно добавить другие компоненты, например, для мониторинга самого Alloy
// prometheus.scrape "self" {
//   targets = prometheus.scrape_sd.alloy_targets(alloy_component_controller.exports.targets)
//   forward_to = [prometheus.remote_write.default.receiver]
// }
// prometheus.remote_write "default" { ... }

```

### Описание параметров конфигурации

Блок `otelcol.exporter.clickhouse "<label>" { ... }`

*   **`dsn`** (string, обязательный):
    *   Строка подключения (Data Source Name) к ClickHouse.
    *   Пример: `"tcp://localhost:9000/default?read_timeout=20s&write_timeout=20s&dial_timeout=5s"`
    *   Включает имя пользователя и пароль, если требуется: `"tcp://user:password@localhost:9000/default"`

*   **`timeout`** (string, опциональный):
    *   Общий таймаут для операций с ClickHouse (например, `"5s"`, `"1m"`).
    *   Если не указан, используется значение по умолчанию из нижележащего экспортера `clickhouseexporter`.

*   **`logs`** (блок, опциональный):
    *   Настройки для экспорта логов.
    *   `table` (string, опциональный): Имя таблицы для логов. По умолчанию: `"otel_logs"`.
    *   `ttl_days` (int, опциональный): Время жизни (TTL) для логов в днях. `0` означает отсутствие TTL. По умолчанию: `0`.

*   **`traces`** (блок, опциональный):
    *   Настройки для экспорта трассировок.
    *   `table` (string, опциональный): Имя таблицы для трассировок. По умолчанию: `"otel_traces"`.
    *   `ttl_days` (int, опциональный): Время жизни (TTL) для трассировок в днях. `0` означает отсутствие TTL. По умолчанию: `0`.

*   **`metrics`** (блок, опциональный):
    *   Настройки для экспорта метрик.
    *   `table` (string, опциональный): Базовое имя таблицы для метрик (например, `"otel_metrics"`). Экспортер автоматически добавит суффиксы для различных типов метрик (например, `_gauge`, `_sum`, `_histogram`). По умолчанию: `"otel_metrics"`.
    *   `ttl_days` (int, опциональный): Время жизни (TTL) для метрик в днях. `0` означает отсутствие TTL. По умолчанию: `0`.

*   **`retry_on_failure`** (блок, опциональный):
    *   Настройки повторных попыток при сбое отправки данных. Соответствует `otelcol.RetryArguments`.
    *   `enabled` (bool, опциональный): Включить повторные попытки. По умолчанию: `true`.
    *   `initial_interval` (duration, опциональный): Начальный интервал между попытками. По умолчанию: `"5s"`.
    *   `max_interval` (duration, опциональный): Максимальный интервал между попытками. По умолчанию: `"30s"`.
    *   `max_elapsed_time` (duration, опциональный): Максимальное общее время, затраченное на повторные попытки. По умолчанию: `"5m"`.

*   **`sending_queue`** (блок, опциональный):
    *   Настройки внутренней очереди для отправки данных. Соответствует `otelcol.QueueSettings`.
    *   `enabled` (bool, опциональный): Включить очередь. По умолчанию: `true`.
    *   `num_consumers` (int, опциональный): Количество обработчиков, отправляющих данные из очереди. По умолчанию: `10`.
    *   `queue_size` (int, опциональный): Размер очереди. По умолчанию: `1000`.

*   **`clickhouse`** (блок, опциональный):
    *   Специфичные настройки для ClickHouse.
    *   `cluster_name` (string, опциональный): Имя кластера ClickHouse, если используется. По умолчанию: `""` (пусто).
    *   `table_engine` (string, опциональный): Движок таблицы, используемый для создания таблиц в ClickHouse (например, `"MergeTree"`, `"ReplicatedMergeTree('/clickhouse/tables/{shard}/{database}/{table}', '{replica}')"`). Если не указан, используется движок по умолчанию из `clickhouseexporter`. По умолчанию: `""` (пусто).

*   **`forward_to`** (атрибут, обязательный для экспортеров):
    *   Список компонентов, которым будут передаваться данные после обработки этим экспортером. В случае экспортера, это обычно пустой список `[]`, так как экспортер является конечной точкой конвейера. Однако, для соответствия модели компонентов Alloy, он должен быть объявлен. В коде компонента `otelcol.exporter.clickhouse` этот аргумент используется для регистрации потребителей, если это необходимо для внутренней логики Alloy, но фактически данные не "пересылаются" дальше, а экспортируются в ClickHouse.

## 5. Важные замечания

*   **Зависимости:** Убедитесь, что ваша среда выполнения Grafana Alloy имеет доступ к серверу ClickHouse, указанному в DSN.
*   **Создание таблиц:** Экспортер попытается автоматически создать необходимые таблицы в ClickHouse при первом запуске, если они не существуют. Для этого у пользователя ClickHouse, указанного в DSN, должны быть соответствующие права (CREATE TABLE).
*   **Производительность:** Для оптимальной производительности рекомендуется использовать `otelcol.processor.batch` перед экспортером `clickhouse`, как указано в документации `opentelemetry-collector-contrib/clickhouseexporter`. Это позволяет отправлять данные в ClickHouse более крупными пакетами.
*   **Версии:** Совместимость компонента зависит от версий Grafana Alloy, OpenTelemetry Collector и `clickhouseexporter`. Следите за обновлениями и соответствующими журналами изменений.
*   **Логирование и мониторинг:** Используйте стандартные механизмы логирования и мониторинга Grafana Alloy для отслеживания работы компонента и диагностики проблем.

## 6. Обратная связь

Поскольку это community component, обратная связь, сообщения об ошибках и предложения по улучшению приветствуются. Пожалуйста, создавайте Issues в соответствующем репозитории Grafana Alloy или в репозитории, где размещен компонент, если он разрабатывается отдельно.
