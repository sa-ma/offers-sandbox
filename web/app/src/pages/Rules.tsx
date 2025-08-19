import React, { useEffect, useState } from 'react'
import { Offer, createOffer, listOffers, updateOffer } from '../api'

export default function Rules() {
  const [offers, setOffers] = useState<Offer[]>([])
  const [form, setForm] = useState({ name: '10% bonus >= £10', minStake: 10, bonusPct: 0.10, active: true })
  const [editing, setEditing] = useState<number | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string|undefined>()

  const load = async () => {
    try { setOffers(await listOffers()) } catch (e:any) { setError(e.message) }
  }
  useEffect(() => { load() }, [])

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError(undefined)
    try {
      if (editing) {
        await updateOffer(editing, form)
      } else {
        await createOffer(form)
      }
      setForm({ name: '10% bonus >= £10', minStake: 10, bonusPct: 0.10, active: true })
      setEditing(null)
      await load()
    } catch (e:any) {
      setError(e.message)
    } finally { setLoading(false) }
  }

  return (
    <div className="space-y-6">
      <h2 className="text-lg font-semibold">Rules</h2>
      <form onSubmit={submit} className="bg-white shadow rounded-md p-4 max-w-xl space-y-4">
        <div>
          <label className="block text-sm font-medium text-gray-700">Name</label>
          <input className="mt-1 block w-full rounded-md border-gray-300 focus:border-gray-900 focus:ring-gray-900" placeholder="Name" value={form.name} onChange={e=>setForm({...form, name: e.target.value})} />
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">Min Stake</label>
            <input className="mt-1 block w-full rounded-md border-gray-300 focus:border-gray-900 focus:ring-gray-900" type="number" step="0.01" value={form.minStake} onChange={e=>setForm({...form, minStake: parseFloat(e.target.value)})} />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">Bonus %</label>
            <input className="mt-1 block w-full rounded-md border-gray-300 focus:border-gray-900 focus:ring-gray-900" type="number" step="0.01" value={form.bonusPct} onChange={e=>setForm({...form, bonusPct: parseFloat(e.target.value)})} />
          </div>
          <div className="flex items-end">
            <label className="inline-flex items-center gap-2">
              <input className="rounded border-gray-300 text-gray-900 focus:ring-gray-900" type="checkbox" checked={form.active} onChange={e=>setForm({...form, active: e.target.checked})} />
              <span>Active</span>
            </label>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <button disabled={loading} type="submit" className="inline-flex items-center rounded-md bg-gray-900 px-4 py-2 text-white hover:bg-black disabled:opacity-50">
            {editing ? 'Update' : 'Create'} Offer
          </button>
          {error && <div className="text-sm text-red-600">{error}</div>}
        </div>
      </form>

      <div>
        <h3 className="text-base font-medium mb-3">Existing Offers</h3>
        <div className="overflow-hidden rounded-md border border-gray-200 bg-white">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                {['ID','Name','Min Stake','Bonus %','Active',''].map((h)=>(
                  <th key={h} className="px-4 py-2 text-left text-xs font-medium uppercase tracking-wider text-gray-500">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {offers.map(o => (
                <tr key={o.id} className="hover:bg-gray-50">
                  <td className="px-4 py-2 text-sm text-gray-700">{o.id}</td>
                  <td className="px-4 py-2 text-sm text-gray-700">{o.name}</td>
                  <td className="px-4 py-2 text-sm text-gray-700">£{o.minStake.toFixed(2)}</td>
                  <td className="px-4 py-2 text-sm text-gray-700">{(o.bonusPct*100).toFixed(0)}%</td>
                  <td className="px-4 py-2 text-sm">{o.active ? <span className="inline-flex items-center rounded-full bg-green-100 px-2 py-0.5 text-green-800">Yes</span> : <span className="inline-flex items-center rounded-full bg-gray-200 px-2 py-0.5 text-gray-700">No</span>}</td>
                  <td className="px-4 py-2 text-right">
                    <button className="text-sm text-gray-900 hover:underline" onClick={()=>{ setEditing(o.id); setForm({ name: o.name, minStake: o.minStake, bonusPct: o.bonusPct, active: o.active }) }}>Edit</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  )
}
