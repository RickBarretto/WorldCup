defmodule Pubsub.Application do
  @moduledoc false

  use Application

  @impl true
  def start(_type, _args) do
    children = [
      # Registry for managing subscribers
      {Registry, keys: :duplicate, name: Pubsub.Registry},
      # Event bus for publishing events
      Pubsub.EventBus,
      # Publisher process
      Pubsub.Publisher,
      # Subscriber processes
      Pubsub.MarkdownSubscriber,
      Pubsub.HtmlSubscriber
    ]

    opts = [strategy: :one_for_one, name: Pubsub.Supervisor]
    Supervisor.start_link(children, opts)
  end
end