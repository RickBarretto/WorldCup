defmodule Pubsub do
  @moduledoc """
  Main module for the Pubsub application.

  This application demonstrates a publisher-subscriber pattern in Elixir
  where articles are published and multiple subscribers handle them differently.
  """

  @doc """
  Start the application manually (useful for testing).
  """
  def start do
    {:ok, _} = Application.ensure_all_started(:pubsub)
  end

  @doc """
  Publish a new article.
  """
  def publish_article(title, content, author) do
    article = Pubsub.Article.new(title, content, author)
    Pubsub.Publisher.publish_article(article)
  end

  @doc """
  Publish the hello world article.
  """
  def publish_hello_world do
    Pubsub.Publisher.publish_hello_world()
  end
end
