{:ok, _} = Application.ensure_all_started(:pubsub)

# Wait long enough for the publisher to auto-publish
:timer.sleep(3000)

IO.puts("Demo complete. Check the generated article files in the current directory.")
