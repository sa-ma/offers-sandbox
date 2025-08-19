export const API_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080'
export const WS_URL = import.meta.env.VITE_WS_URL || 'ws://localhost:8083/ws'

export type Offer = {
  id: number
  name: string
  minStake: number
  bonusPct: number
  active: boolean
  createdAt: string
}

export type Award = {
  awardId: string
  eventId: string
  userId: string
  offerId: number
  stake: number
  bonusAmount: number
  ts?: number
  createdAt?: string
}

export async function listOffers(): Promise<Offer[]> {
  const r = await fetch(`${API_URL}/offers`)
  if (!r.ok) throw new Error('Failed to load offers')
  return r.json()
}

export async function createOffer(p: {name:string, minStake:number, bonusPct:number, active:boolean}) {
  const r = await fetch(`${API_URL}/offers`, {
    method: 'POST', headers: {'Content-Type':'application/json'}, body: JSON.stringify(p)
  })
  if (!r.ok) throw new Error('Failed to create offer')
  return r.json()
}

export async function updateOffer(id:number, p: {name:string, minStake:number, bonusPct:number, active:boolean}) {
  const r = await fetch(`${API_URL}/offers/${id}`, {
    method: 'PUT', headers: {'Content-Type':'application/json'}, body: JSON.stringify(p)
  })
  if (!r.ok) throw new Error('Failed to update offer')
  return r.json()
}
