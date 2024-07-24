<script>
  import { state } from "../lib/state";
  import * as runtime from "../../wailsjs/runtime";

  let st;
  state.subscribe((x) => {
    st = x;
  });

  runtime.EventsOn("peer-connect", (peer) => {
    console.log(peer);
    const peerEl = document.createElement("p");
    const peerLine = document.createTextNode(peer.ip + ":" + peer.port);
    peerEl.appendChild(peerLine);

    const peerList = document.getElementById("peers");
    peerList.appendChild(peerEl);
  });

  runtime.EventsOn("state-update", (s) => {
    state.set(s);
  });
</script>

<main>
  <div>
    <h1>state</h1>
  </div>
  <div>
    <p>{st}</p>
  </div>
  <h1>peers</h1>
  <div id="peers"></div>
</main>

<style>
</style>
