'use client'

import { useEffect, useState } from 'react'

interface Event {
  id: number
  component: string
  eventType: string
  command?: string
  operatorRole?: string
  decision?: string
  reason?: string
  status?: string
  message?: string
  createdAt: string
}

export default function Home() {
  const [events, setEvents] = useState<Event[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchEvents = async () => {
    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const response = await fetch(`${apiUrl}/api/v1/events?limit=50`)
      if (!response.ok) {
        throw new Error('無法載入事件')
      }
      const data = await response.json()
      setEvents(data.events || [])
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : '未知錯誤')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchEvents()
    // 每 5 秒自動重新載入
    const interval = setInterval(fetchEvents, 5000)
    return () => clearInterval(interval)
  }, [])

  const formatTime = (timeStr: string) => {
    return new Date(timeStr).toLocaleString('zh-TW')
  }

  const getDecisionColor = (decision?: string) => {
    if (decision === 'allowed') return 'text-green-600'
    if (decision === 'denied') return 'text-red-600'
    return 'text-gray-600'
  }

  return (
    <main className="min-h-screen p-8 bg-gray-50">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-3xl font-bold mb-6 text-gray-900">Space-SOC Dashboard</h1>
        
        <div className="mb-4 flex gap-4 items-center">
          <button
            onClick={fetchEvents}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
          >
            重新載入
          </button>
          <div className="text-sm text-gray-600">
            總事件數: <span className="font-semibold">{events.length}</span>
          </div>
        </div>

        {loading && <div className="text-center py-8 text-gray-600">載入中...</div>}
        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
            錯誤: {error}
          </div>
        )}

        {!loading && !error && (
          <div className="overflow-x-auto bg-white rounded-lg shadow">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-100">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 uppercase tracking-wider">時間</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 uppercase tracking-wider">組件</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 uppercase tracking-wider">事件類型</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 uppercase tracking-wider">指令</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 uppercase tracking-wider">操作者角色</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 uppercase tracking-wider">決策</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 uppercase tracking-wider">狀態</th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 uppercase tracking-wider">訊息</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {events.length === 0 ? (
                  <tr>
                    <td colSpan={8} className="px-4 py-8 text-center text-gray-500">
                      尚無事件
                    </td>
                  </tr>
                ) : (
                  events.map((event) => (
                    <tr key={event.id} className="hover:bg-gray-50">
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900">
                        {formatTime(event.createdAt)}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm font-medium text-gray-900">
                        {event.component}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-700">
                        {event.eventType}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-700">
                        {event.command || '-'}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-700">
                        {event.operatorRole || '-'}
                      </td>
                      <td className={`px-4 py-3 whitespace-nowrap text-sm font-semibold ${getDecisionColor(event.decision)}`}>
                        {event.decision || '-'}
                      </td>
                      <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-700">
                        {event.status || '-'}
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-700">
                        {event.message || '-'}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </main>
  )
}
