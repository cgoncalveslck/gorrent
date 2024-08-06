<script>
  import logo from "../assets/images/logo-universal.png";
  import { OpenFileDialog } from "../../wailsjs/go/main/App.js";
  import { push } from "svelte-spa-router";
  import { state } from "../lib/state";

  let resultText = "Choose file below 👇";

  function openFileDialog() {
    OpenFileDialog().then((res) => {
      state.set({
        file: {
          name: res.info.name,
        },
      });
      push("/files");
    });
  }
</script>

<main>
  <img alt="Wails logo" id="logo" src={logo} />
  <div class="result" id="result">{resultText}</div>
  <div class="input-box" id="input">
    <button class="btn" on:click={openFileDialog}>Select file</button>
  </div>
</main>

<style>
  #logo {
    display: block;
    height: 50%;
    margin: auto;
    padding: 10% 0 0;
    background-position: center;
    background-repeat: no-repeat;
    background-size: 100% 100%;
    background-origin: content-box;
  }

  main {
    height: 100vh;
  }

  .result {
    height: 20px;
    line-height: 20px;
    margin: 1.5rem auto;
  }

  .input-box .btn {
    height: 30px;
    line-height: 30px;
    border-radius: 3px;
    border: none;
    margin: 0 0 0 20px;
    padding: 0 8px;
    cursor: pointer;
  }

  .input-box .btn:hover {
    background-image: linear-gradient(to top, #cfd9df 0%, #e2ebf0 100%);
    color: #333333;
  }
</style>
