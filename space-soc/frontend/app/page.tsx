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
  severity?: string
  anomalyType?: string
  scenarioID?: string
  createdAt: string
}

interface Incident {
  id: number
  title: string
  description: string
  severity: string
  status: string
  scenarioID?: string
  events?: Event[]
  createdAt: string
  updatedAt: string
}

export default function Home() {
  const [events, setEvents] = useState<Event[]>([])
  const [incidents, setIncidents] = useState<Incident[]>([])
  const [activeTab, setActiveTab] = useState<'events' | 'incidents'>('events')
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

  const fetchIncidents = async () => {
    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'
      const response = await fetch(`${apiUrl}/api/v1/incidents`)
      if (!response.ok) {
        throw new Error('無法載入 incidents')
      }
      const data = await response.json()
      setIncidents(data.incidents || [])
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : '未知錯誤')
    }
  }

  useEffect(() => {
    fetchEvents()
    fetchIncidents()
    // 每 5 秒自動重新載入
    const interval = setInterval(() => {
      fetchEvents()
      fetchIncidents()
    }, 5000)
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

  const getSeverityColor = (severity?: string) => {
    switch (severity) {
      case 'critical': return 'bg-red-100 text-red-800 border-red-300'
      case 'high': return 'bg-orange-100 text-orange-800 border-orange-300'
      case 'medium': return 'bg-yellow-100 text-yellow-800 border-yellow-300'
      case 'low': return 'bg-blue-100 text-blue-800 border-blue-300'
      default: return 'bg-gray-100 text-gray-800 border-gray-300'
    }
  }

  const getStatusColor = (status?: string) => {
    switch (status) {
      case 'open': return 'bg-red-100 text-red-800'
      case 'investigating': return 'bg-yellow-100 text-yellow-800'
      case 'resolved': return 'bg-green-100 text-green-800'
      case 'closed': return 'bg-gray-100 text-gray-800'
      default: return 'bg-gray-100 text-gray-800'
    }
  }

  return (
    <main className="min-h-screen p-8 bg-gray-50">
      <div className="max-w-7xl mx-auto">
        <h1 className="text-3xl font-bold mb-6 text-gray-900">Space-SOC Dashboard</h1>

        {/* 標籤切換 */}
        <div className="mb-4 flex gap-2 border-b border-gray-200">
          <button
            onClick={() => setActiveTab('events')}
            className={`px-4 py-2 font-medium ${
              activeTab === 'events'
                ? 'border-b-2 border-blue-600 text-blue-600'
                : 'text-gray-600 hover:text-gray-900'
            }`}
          >
            事件 ({events.length})
          </button>
          <button
            onClick={() => setActiveTab('incidents')}
            className={`px-4 py-2 font-medium ${
              activeTab === 'incidents'
                ? 'border-b-2 border-blue-600 text-blue-600'
                : 'text-gray-600 hover:text-gray-900'
            }`}
          >
            安全事件 ({incidents.length})
          </button>
        </div>

        <div className="mb-4 flex gap-4 items-center">
          <button
            onClick={() => {
              fetchEvents()
              fetchIncidents()
            }}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
          >
            重新載入
          </button>
          {activeTab === 'events' && (
            <div className="text-sm text-gray-600">
              總事件數: <span className="font-semibold">{events.length}</span>
            </div>
          )}
          {activeTab === 'incidents' && (
            <div className="text-sm text-gray-600">
              開放事件: <span className="font-semibold">
                {incidents.filter(i => i.status === 'open' || i.status === 'investigating').length}
              </span>
            </div>
          )}
        </div>

        {loading && <div className="text-center py-8 text-gray-600">載入中...</div>}
        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
            錯誤: {error}
          </div>
        )}

        {!loading && !error && (
          <>
            {activeTab === 'events' && (
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
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 uppercase tracking-wider">嚴重性</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 uppercase tracking-wider">異常類型</th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-700 uppercase tracking-wider">訊息</th>
                    </tr>
                  </thead>
                  <tbody className="bg-white divide-y divide-gray-200">
                    {events.length === 0 ? (
                      <tr>
                        <td colSpan={9} className="px-4 py-8 text-center text-gray-500">
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
                          <td className="px-4 py-3 whitespace-nowrap">
                            {event.severity && (
                              <span className={`px-2 py-1 text-xs font-semibold rounded border ${getSeverityColor(event.severity)}`}>
                                {event.severity}
                              </span>
                            )}
                            {!event.severity && '-'}
                          </td>
                          <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-700">
                            {event.anomalyType || '-'}
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

            {activeTab === 'incidents' && (
              <div className="space-y-4">
                {incidents.length === 0 ? (
                  <div className="bg-white rounded-lg shadow p-8 text-center text-gray-500">
                    尚無安全事件
                  </div>
                ) : (
                  incidents.map((incident) => (
                    <div key={incident.id} className="bg-white rounded-lg shadow p-6">
                      <div className="flex items-start justify-between mb-4">
                        <div>
                          <h3 className="text-lg font-semibold text-gray-900">{incident.title}</h3>
                          <p className="text-sm text-gray-600 mt-1">{incident.description}</p>
                        </div>
                        <div className="flex gap-2">
                          <span className={`px-3 py-1 text-xs font-semibold rounded ${getSeverityColor(incident.severity)}`}>
                            {incident.severity}
                          </span>
                          <span className={`px-3 py-1 text-xs font-semibold rounded ${getStatusColor(incident.status)}`}>
                            {incident.status}
                          </span>
                        </div>
                      </div>
                      <div className="text-xs text-gray-500">
                        創建時間: {formatTime(incident.createdAt)} | 
                        更新時間: {formatTime(incident.updatedAt)}
                        {incident.scenarioID && ` | 場景: ${incident.scenarioID}`}
                      </div>
                      {incident.events && incident.events.length > 0 && (
                        <div className="mt-4 pt-4 border-t border-gray-200">
                          <p className="text-sm font-medium text-gray-700 mb-2">
                            相關事件 ({incident.events.length})
                          </p>
                          <div className="space-y-1">
                            {incident.events.slice(0, 3).map((event) => (
                              <div key={event.id} className="text-xs text-gray-600">
                                • {event.eventType} - {event.component} - {formatTime(event.createdAt)}
                              </div>
                            ))}
                            {incident.events.length > 3 && (
                              <div className="text-xs text-gray-500">
                                ... 還有 {incident.events.length - 3} 個事件
                              </div>
                            )}
                          </div>
                        </div>
                      )}
                    </div>
                  ))
                )}
              </div>
            )}
          </>
        )}
      </div>
    </main>
  )
}
