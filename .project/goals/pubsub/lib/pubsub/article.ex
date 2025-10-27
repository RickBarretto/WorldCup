defmodule Pubsub.Article do
  @moduledoc """
  Article structure for the publisher-subscriber system.
  """

  @derive Jason.Encoder
  defstruct [:id, :title, :content, :author, :published_at]

  @type t :: %__MODULE__{
          id: String.t(),
          title: String.t(),
          content: String.t(),
          author: String.t(),
          published_at: DateTime.t()
        }

  @doc """
  Creates a new article.
  """
  def new(title, content, author) do
    %__MODULE__{
      id: generate_id(),
      title: title,
      content: content,
      author: author,
      published_at: DateTime.utc_now()
    }
  end

  defp generate_id do
    :crypto.strong_rand_bytes(16)
    |> Base.encode16(case: :lower)
  end

  @doc """
  Creates a sample "Hello World" article.
  """
  def hello_world do
    %__MODULE__{
      id: "hello-world-001",
      title: "Hello World",
      content: "This is a hello world article for our publisher-subscriber system. Welcome to the world of Elixir pub-sub patterns!",
      author: "Elixir Publisher",
      published_at: DateTime.utc_now()
    }
  end

  @doc """
  Convert article to JSON.
  """
  def to_json(article) do
    Jason.encode!(article)
  end
end