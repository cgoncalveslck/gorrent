import { writable } from "svelte/store";

function createState(value) {
    const { subscribe, set, update } = writable(value)


    return { subscribe, set, update }
}

export const state = createState({
  torrents: []
})