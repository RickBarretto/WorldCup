defmodule Pubsub.Publisher do
  @moduledoc """
  Publisher process that publishes articles to the event bus.
  """

  use GenServer
  require Logger

  def start_link(_opts) do
    GenServer.start_link(__MODULE__, %{}, name: __MODULE__)
  end

  @doc """
  Publish an article to all subscribers.
  """
  def publish_article(article) do
    GenServer.cast(__MODULE__, {:publish_article, article})
  end

  @doc """
  Publish the hello world article.
  """
  def publish_hello_world do
    article = Pubsub.Article.hello_world()
    publish_article(article)
  end

  @impl true
  def init(state) do
    Logger.info("Publisher started")
    Process.send_after(self(), :publish_hello_world, 2000)
    {:ok, state}
  end

  @impl true
  def handle_cast({:publish_article, article}, state) do
    Logger.info("Publishing article: #{article.title}")
    Pubsub.EventBus.publish(:article_published, article)
    {:noreply, state}
  end

  @impl true
  def handle_info(:publish_hello_world, state) do
    Logger.info("Auto-publishing Hello World article")
    publish_hello_world()
    {:noreply, state}
  end
end
