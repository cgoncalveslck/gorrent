<script>
  import Router from "svelte-spa-router";

  import Landing from "./routes/Landing.svelte";
  import Files from "./routes/Files.svelte";

  const routes = {
    "/": Landing,
    "/files": Files,
  };
</script>

<body>
  <Router {routes} />
</body>
