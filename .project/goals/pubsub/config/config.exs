import Config

# Configure logger
config :logger, :console,
  level: :info,
  format: "$date $time [$level] $metadata$message\n"

# Import environment specific config
import_config "#{config_env()}.exs"