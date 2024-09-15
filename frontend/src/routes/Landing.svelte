<script>
  import { writable, derived } from 'svelte/store';
  import { state } from "../lib/state";
  import { OpenFileDialog, GetDevTorrent } from "../../wailsjs/go/main/App.js";
  import { Search, Download, Pause, Play, Trash2, Info, X } from 'lucide-svelte';

  let torrentsStore = writable([]);
  let searchQuery = writable("");
  let sortBy = writable("name");
  let sortOrder = writable("asc");
  let selectedTorrent = writable(null);

  state.subscribe(x => {
    console.log("State updated:", x);
    torrentsStore.set(x.torrents || []);
  });

  const filteredAndSortedTorrents = derived(
    [torrentsStore, searchQuery, sortBy, sortOrder],
    ([$torrents, $searchQuery, $sortBy, $sortOrder]) => {
      console.log("Deriving filteredAndSortedTorrents", { $torrents, $searchQuery, $sortBy, $sortOrder });
      
      let filtered = $torrents.filter(torrent => 
        torrent.torrentName.toLowerCase().includes($searchQuery.toLowerCase())
      );

      filtered.sort((a, b) => {
        let compareResult;
        switch ($sortBy) {
          case "name":
            compareResult = a.torrentName.localeCompare(b.torrentName);
            break;
          case "size":
            compareResult = a.totalLength - b.totalLength;
            break;
          case "progress":
            compareResult = a.progress - b.progress;
            break;
          default:
            compareResult = 0;
        }
        return $sortOrder === "asc" ? compareResult : -compareResult;
      });

      return filtered;
    }
  );

  function openFileDialog() {
    OpenFileDialog().then((res) => {
      torrentsStore.update(currentTorrents => [
        ...currentTorrents,
        {
          ...res,
          id: generateUniqueId(),
          isPaused: false,
        }
      ]);
    });
  }

  function generateUniqueId() {
    return Date.now().toString(36) + Math.random().toString(36).substr(2);
  }

  function addNewTorrent() {
    openFileDialog();
  }

  function openTorrentDetails(torrent) {
    selectedTorrent.set(torrent);
  }

  function closeTorrentDetails(event) {
    if (event && event.target === event.currentTarget) {
      selectedTorrent.set(null);
    }
  }

  function formatSize(bytes) {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    let size = bytes;
    let unitIndex = 0;
    while (size >= 1024 && unitIndex < units.length - 1) {
      size /= 1024;
      unitIndex++;
    }
    return `${size.toFixed(2)} ${units[unitIndex]}`;
  }

  function togglePause(torrent) {
    torrentsStore.update(currentTorrents => 
      currentTorrents.map(t => 
        t.id === torrent.id ? { ...t, isPaused: !t.isPaused } : t
      )
    );
  }

  function removeTorrent(torrentId) {
    torrentsStore.update(currentTorrents => 
      currentTorrents.filter(t => t.id !== torrentId)
    );
  }

  function formatSpeed(bytesPerSecond) {
    if (bytesPerSecond === 0) return '0 B/s';
    const units = ['B/s', 'KB/s', 'MB/s', 'GB/s'];
    const i = Math.floor(Math.log(bytesPerSecond) / Math.log(1024));
    return `${(bytesPerSecond / Math.pow(1024, i)).toFixed(2)} ${units[i]}`;
  }

  function formatETA(seconds) {
    if (seconds === Infinity || seconds === 0) return 'Unknown';
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${minutes}m`;
  }

  function loadDevTorrent() {
    GetDevTorrent().then((res) => {
      torrentsStore.update(currentTorrents => [
        ...currentTorrents,
        {
          ...res,
          id: generateUniqueId(),
          isPaused: false,
        }
      ]);
    });
  }
</script>

<main>
  <h1>Torrent Client</h1>
  
  <div class="controls">
    <div class="search-bar">
      <Search size={20} />
      <input type="text" bind:value={$searchQuery} placeholder="Search torrents...">
    </div>
    <select bind:value={$sortBy}>
      <option value="name">Name</option>
      <option value="size">Size</option>
      <option value="progress">Progress</option>
    </select>
    <button class="sort-order" on:click={() => $sortOrder = $sortOrder === "asc" ? "desc" : "asc"}>
      {$sortOrder === "asc" ? "↑" : "↓"}
    </button>
    <button class="btn primary" on:click={addNewTorrent}>
      <Download size={20} />
      Add New Torrent
    </button>
  </div>

  <div class="torrent-list">
    {#each $filteredAndSortedTorrents as torrent (torrent.id)}
      <div class="torrent-item">
        <div class="torrent-info">
          <h3 class="torrent-name">{torrent.torrentName}</h3>
          <p class="torrent-details">
            {torrent.isMultiFile ? `${torrent.fileNames.length} files` : '1 file'} | 
            {formatSize(torrent.totalLength)}
          </p>
        </div>
        <div class="torrent-progress">
          <div class="progress-bar">
            <div class="progress-fill" style="width: {torrent.progress}%"></div>
          </div>
          <span class="progress-text">{torrent.progress.toFixed(1)}%</span>
        </div>
        <div class="torrent-actions">
          <button class="btn icon" on:click={() => togglePause(torrent)}>
            {#if torrent.isPaused}
              <Play size={20} />
            {:else}
              <Pause size={20} />
            {/if}
          </button>
          <button class="btn icon" on:click={() => openTorrentDetails(torrent)}>
            <Info size={20} />
          </button>
          <button class="btn icon danger" on:click={() => removeTorrent(torrent.id)}>
            <Trash2 size={20} />
          </button>
        </div>
      </div>
    {/each}
  </div>

  {#if $selectedTorrent}
  <div 
    class="modal-overlay" 
    on:click={closeTorrentDetails}
    on:keydown={(e) => e.key === 'Escape' && closeTorrentDetails()}
    tabindex="0"
    role="button"
    aria-label="Close modal"
  >
    <div class="modal-content">
      <button 
        class="close-btn" 
        on:click={closeTorrentDetails}
        on:keydown={(e) => e.key === 'Enter' && closeTorrentDetails()}
      >
        <X size={24} />
      </button>
      <h2>{$selectedTorrent.torrentName}</h2>
      <div class="torrent-details-grid">
        <div class="detail-item">
          <strong>Size:</strong> {formatSize($selectedTorrent.totalLength)}
        </div>
        <div class="detail-item">
          <strong>Progress:</strong> {$selectedTorrent.progress.toFixed(1)}%
        </div>
        <div class="detail-item">
          <strong>Status:</strong> {$selectedTorrent.isPaused ? 'Paused' : 'Downloading'}
        </div>
        <div class="detail-item">
          <strong>Download Speed:</strong> {formatSpeed($selectedTorrent.downloadSpeed)}
        </div>
        <div class="detail-item">
          <strong>Upload Speed:</strong> {formatSpeed($selectedTorrent.uploadSpeed)}
        </div>
        <div class="detail-item">
          <strong>ETA:</strong> {formatETA($selectedTorrent.eta)}
        </div>
        <div class="detail-item">
          <strong>Peers:</strong> {$selectedTorrent.peers}
        </div>
        <div class="detail-item">
          <strong>Seeds:</strong> {$selectedTorrent.seeds}
        </div>
      </div>
      {#if $selectedTorrent.isMultiFile}
        <div class="file-list">
          <h3>Files:</h3>
          <ul>
            {#each $selectedTorrent.fileNames as fileName, index}
              <li>
                {fileName}
                <span class="file-progress">
                  ({($selectedTorrent.fileProgress[index] * 100).toFixed(1)}%)
                </span>
              </li>
            {/each}
          </ul>
        </div>
      {/if}
      <div class="torrent-graph">
        <h3>Download History</h3>
        <!-- Add a placeholder for a graph showing download speed over time -->
        <div class="graph-placeholder">Graph placeholder</div>
      </div>
    </div>
  </div>
  {/if}

  <!-- Hidden button for loading a dev torrent -->
  <button class="btn debug" on:click={loadDevTorrent}>
    Load Dev Torrent
  </button>
</main>

<style>
  :root {
    --background-color: rgba(18, 18, 18, 0.7);
    --text-color: #e0e0e0;
    --accent-color: #2196F3;
    --danger-color: #F44336;
    --item-hover-color: rgba(255, 255, 255, 0.1);
    --border-color: rgba(255, 255, 255, 0.1);
  }

  main {
    height: 100vh;
    padding: 2rem;
    max-width: 1000px;
    margin: 0 auto;
    font-family: Arial, sans-serif;
    color: var(--text-color);
    position: relative; /* Added for positioning the debug button */
  }

  h1 {
    color: var(--accent-color);
    margin-bottom: 2rem;
    font-weight: 300;
    letter-spacing: 1px;
  }

  .controls {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 2rem;
    background-color: var(--background-color);
    padding: 1rem;
    border-radius: 8px;
    backdrop-filter: blur(10px);
  }

  .search-bar {
    display: flex;
    align-items: center;
    background-color: rgba(255, 255, 255, 0.1);
    border-radius: 20px;
    padding: 0.5rem 1rem;
    flex-grow: 1;
    margin-right: 1rem;
  }

  .search-bar input {
    background: transparent;
    border: none;
    outline: none;
    margin-left: 0.5rem;
    font-size: 1rem;
    width: 100%;
    color: var(--text-color);
  }

  .search-bar input::placeholder {
    color: rgba(255, 255, 255, 0.5);
  }

  select {
    padding: 0.5rem;
    border-radius: 5px;
    border: 1px solid var(--border-color);
    background-color: rgba(255, 255, 255, 0.1);
    color: var(--text-color);
    margin-right: 0.5rem;
  }

  .sort-order {
    padding: 0.5rem 1rem;
    border: 1px solid var(--border-color);
    background-color: rgba(255, 255, 255, 0.1);
    color: var(--text-color);
    border-radius: 5px;
    cursor: pointer;
    margin-right: 1rem;
  }

  .btn {
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 5px;
    cursor: pointer;
    font-size: 1rem;
    display: flex;
    align-items: center;
    gap: 0.5rem;
    background-color: rgba(255, 255, 255, 0.1);
    color: var(--text-color);
    transition: background-color 0.3s ease;
  }

  .btn:hover {
    background-color: rgba(255, 255, 255, 0.2);
  }

  .btn.primary {
    background-color: var(--accent-color);
    color: white;
  }

  .btn.primary:hover {
    background-color: #1976D2;
  }

  .btn.icon {
    padding: 0.5rem;
    background-color: transparent;
  }

  .btn.danger {
    color: var(--danger-color);
  }

  .torrent-list {
    background-color: var(--background-color);
    border-radius: 8px;
    overflow: hidden;
    backdrop-filter: blur(10px);
  }

  .torrent-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 1rem;
    border-bottom: 1px solid var(--border-color);
    transition: background-color 0.2s ease;
  }

  .torrent-item:last-child {
    border-bottom: none;
  }

  .torrent-item:hover {
    background-color: var(--item-hover-color);
  }

  .torrent-info {
    flex-grow: 1;
  }

  .torrent-name {
    margin: 0;
    font-size: 1.1rem;
    font-weight: 500;
  }

  .torrent-details {
    margin: 0.25rem 0 0;
    font-size: 0.9rem;
    color: rgba(255, 255, 255, 0.7);
  }

  .torrent-progress {
    display: flex;
    align-items: center;
    min-width: 200px;
    margin: 0 1rem;
  }

  .progress-bar {
    width: 100%;
    height: 4px;
    background-color: rgba(255, 255, 255, 0.2);
    border-radius: 2px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background-color: var(--accent-color);
    transition: width 0.3s ease;
  }

  .progress-text {
    font-size: 0.9rem;
    color: rgba(255, 255, 255, 0.7);
    margin-left: 1rem;
    min-width: 50px;
    text-align: right;
  }

  .torrent-actions {
    display: flex;
    gap: 0.5rem;
  }

  .modal-overlay {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
  }

  .modal-content {
    background-color: var(--background-color);
    padding: 2rem;
    border-radius: 8px;
    max-width: 80%;
    max-height: 80%;
    overflow-y: auto;
    position: relative;
  }

  .close-btn {
    position: absolute;
    top: 1rem;
    right: 1rem;
    background: none;
    border: none;
    color: var(--text-color);
    cursor: pointer;
  }

  .torrent-details-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 1rem;
    margin-top: 1rem;
  }

  .detail-item {
    background-color: rgba(255, 255, 255, 0.1);
    padding: 0.5rem;
    border-radius: 4px;
  }

  .file-list {
    margin-top: 1rem;
  }

  .file-list ul {
    list-style-type: none;
    padding-left: 0;
    max-height: 200px;
    overflow-y: auto;
  }

  .file-list li {
    padding: 0.25rem 0;
  }

  .modal-content {
    width: 80%;
    max-width: 800px;
    max-height: 90vh;
    overflow-y: auto;
  }

  .file-list {
    margin-top: 1rem;
    max-height: 200px;
    overflow-y: auto;
  }

  .file-list ul {
    list-style-type: none;
    padding-left: 0;
  }

  .file-list li {
    padding: 0.25rem 0;
    display: flex;
    justify-content: space-between;
  }

  .file-progress {
    color: var(--accent-color);
  }

  .torrent-graph {
    margin-top: 1rem;
  }

  .graph-placeholder {
    background-color: rgba(255, 255, 255, 0.1);
    height: 200px;
    display: flex;
    justify-content: center;
    align-items: center;
    border-radius: 4px;
  }

  .btn.debug {
    position: fixed;
    bottom: 1rem;
    right: 1rem;
    background-color: rgba(255, 255, 255, 0.1);
    color: var(--text-color);
    opacity: 0.5;
    transition: opacity 0.3s ease;
  }

  .btn.debug:hover {
    opacity: 1;
  }
</style>