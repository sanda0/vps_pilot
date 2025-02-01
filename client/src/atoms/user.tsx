import {atom} from 'jotai'

export interface User {
  email: string;
  token: string;
  username: string;
}

export const userAtom = atom<User | null>(null)