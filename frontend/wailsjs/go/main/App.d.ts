// Cynhyrchwyd y ffeil hon yn awtomatig. PEIDIWCH Â MODIWL
// This file is automatically generated. DO NOT EDIT
import {backend} from '../models';

export function AddTorrent(arg1:backend.Torrent):Promise<void>;

export function GetDevTorrent():Promise<backend.Torrent>;

export function GetTorrents():Promise<Array<backend.Torrent>>;

export function OpenFileDialog():Promise<backend.Torrent>;

export function RemoveTorrent(arg1:number):Promise<void>;
