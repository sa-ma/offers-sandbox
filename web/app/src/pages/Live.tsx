import React, { useEffect, useRef, useState } from 'react'
import { WS_URL, Award } from '../api'

export default function Live() {
  const [awards, setAwards] = useState<Award[]>([])
  const socketRef = useRef<WebSocket | null>(null)

  useEffect(() => {
    const ws = new WebSocket(WS_URL)
    socketRef.current = ws
    ws.onmessage = (ev) => {
      try {
        const a: Award = JSON.parse(ev.data)
        setAwards(prev => [a, ...prev].slice(0, 200))
      } catch {}
    }
    ws.onopen = () => console.log('WS connected')
    ws.onclose = () => console.log('WS closed')
    return () => ws.close()
  }, [])

  return (
    <div className="space-y-4">
      <h2 className="text-lg font-semibold">Live Awards</h2>
      <div className="overflow-hidden rounded-md border border-gray-200 bg-white">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              {['Time','User','Offer','Stake','Bonus'].map((h)=>(
                <th key={h} className="px-4 py-2 text-left text-xs font-medium uppercase tracking-wider text-gray-500">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {awards.map(a => (
              <tr key={a.awardId} className="hover:bg-gray-50">
                <td className="px-4 py-2 text-sm text-gray-700">{new Date(a.ts ?? Date.now()).toLocaleTimeString()}</td>
                <td className="px-4 py-2 text-sm text-gray-700">{a.userId}</td>
                <td className="px-4 py-2 text-sm text-gray-700">{a.offerId}</td>
                <td className="px-4 py-2 text-sm text-gray-700">£{a.stake.toFixed(2)}</td>
                <td className="px-4 py-2 text-sm text-gray-900 font-medium">£{a.bonusAmount.toFixed(2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}
