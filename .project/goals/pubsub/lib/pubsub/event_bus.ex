defmodule Pubsub.EventBus do
  @moduledoc """
  Event bus for managing publisher-subscriber communication.
  """

  use GenServer

  def start_link(_opts) do
    GenServer.start_link(__MODULE__, %{}, name: __MODULE__)
  end

  @doc """
  Subscribe to events of a specific type.
  """
  def subscribe(event_type) do
    Registry.register(Pubsub.Registry, event_type, [])
  end

  @doc """
  Publish an event to all subscribers of that event type.
  """
  def publish(event_type, event_data) do
    GenServer.cast(__MODULE__, {:publish, event_type, event_data})
  end

  @impl true
  def init(state) do
    {:ok, state}
  end

  @impl true
  def handle_cast({:publish, event_type, event_data}, state) do
    # Find all subscribers for this event type
    Registry.dispatch(Pubsub.Registry, event_type, fn subscribers ->
      for {pid, _} <- subscribers do
        send(pid, {:event, event_type, event_data})
      end
    end)

    {:noreply, state}
  end
end
