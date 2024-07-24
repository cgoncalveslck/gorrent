<script>
  import { state } from "../lib/state";
  import { get } from "svelte/store";
  import { onMount } from "svelte";
  import * as runtime from "../../wailsjs/runtime";

  let peerList;
  let torrent = get(state).file.name;

  function updatePeerStatus(peer, status) {
    const { port, ip } = peer;
    if (!port || !ip) return;

    const peerId = `peer-${ip.replace(/[^a-zA-Z0-9]/g, "")}-${port}`;
    let peerEl = peerList.querySelector(`#${peerId}`);

    if (!peerEl) {
      peerEl = document.createElement("div");
      peerEl.id = peerId;
      peerEl.className = "peer";

      const peerLine = document.createElement("span");
      peerLine.textContent = `${ip}:${port}`;
      peerEl.appendChild(peerLine);

      const statusIndicator = document.createElement("span");
      statusIndicator.className = "status-indicator";
      peerEl.appendChild(statusIndicator);

      peerList.appendChild(peerEl);
    }

    const statusIndicator = peerEl.querySelector(".status-indicator");
    if (statusIndicator) {
      statusIndicator.style.backgroundColor =
        status === "connected" ? "green" : "red";
    }
  }

  onMount(() => {
    runtime.EventsOn("peer-connect", (peer) => {
      updatePeerStatus(peer, "connected");
      state.set({ peer });
    });

    runtime.EventsOn("peer-disconnect", (peer) => {
      updatePeerStatus(peer, "disconnected");
    });

    runtime.EventsOn("state-update", (s) => {
      // not being used
      console.log("backend state update", s);

      state.set(s);
    });
  });
</script>

<main>
  <div>
    <h1>Torrrent Info</h1>
    <div>
      <p>File: {torrent}</p>
    </div>
  </div>
  <div>
    <h1>peers</h1>
    <div id="peers" bind:this={peerList}></div>
  </div>
  <div></div>
</main>

<style>
  main {
    display: flex;
    flex-direction: column;
  }

  #peers {
    display: flex;
    flex-wrap: wrap;
    justify-content: start;
    gap: 22px 0;
    margin: 0 40px;
  }

  :global(.peer) {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    width: 20%;
  }

  :global(.status-indicator) {
    display: inline-block;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    margin-left: 10px;
  }
</style>
