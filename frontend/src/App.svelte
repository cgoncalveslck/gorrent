<script>
  import Router from "svelte-spa-router";

  import Landing from "./routes/Landing.svelte";

  const routes = {
    "/": Landing,
  };
</script>

<body>
  <Router {routes} />
</body>
