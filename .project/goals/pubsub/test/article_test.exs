defmodule Pubsub.ArticleTest do
  use ExUnit.Case, async: true
  
  alias Pubsub.Article

  describe "Article" do
    test "creates a new article with all required fields" do
      article = Article.new("Test Title", "Test content", "Test Author")
      
      assert article.title == "Test Title"
      assert article.content == "Test content"
      assert article.author == "Test Author"
      assert is_binary(article.id)
      assert %DateTime{} = article.published_at
    end

    test "creates hello world article" do
      article = Article.hello_world()
      
      assert article.title == "Hello World"
      assert article.author == "Elixir Publisher"
      assert article.id == "hello-world-001"
      assert String.contains?(article.content, "hello world")
    end

    test "converts article to JSON" do
      article = Article.hello_world()
      json_string = Article.to_json(article)
      
      assert is_binary(json_string)
      assert String.contains?(json_string, "Hello World")
    end
  end
end