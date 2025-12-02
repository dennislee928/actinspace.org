'use client'

import { useEffect, useState } from 'react'
import Link from 'next/link'

interface SoftwarePosture {
  id: number
  component: string
  currentVersion: string
  latestVersion?: string
  imageDigest?: string
  sbomUrl?: string
  vulnCount: number
  lastScanTime?: string
  lastUpdateTime?: string
  updateAvailable: boolean
  createdAt: string
  updatedAt: string
}

export default function PosturePage() {
  const [postures, setPostures] = useState<SoftwarePosture[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchPostures = async () => {
    try {
      const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8083'
      const response = await fetch(`${apiUrl}/api/v1/posture`)
      if (!response.ok) {
        throw new Error('無法載入軟體姿態')
      }
      const data = await response.json()
      setPostures(data.postures || [])
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : '未知錯誤')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchPostures()
    // 每 10 秒自動重新載入
    const interval = setInterval(fetchPostures, 10000)
    return () => clearInterval(interval)
  }, [])

  const formatTime = (timeStr?: string) => {
    if (!timeStr) return '-'
    return new Date(timeStr).toLocaleString('zh-TW')
  }

  const getVulnColor = (count: number) => {
    if (count === 0) return 'bg-green-100 text-green-800'
    if (count <= 5) return 'bg-yellow-100 text-yellow-800'
    if (count <= 10) return 'bg-orange-100 text-orange-800'
    return 'bg-red-100 text-red-800'
  }

  return (
    <main className="min-h-screen p-8 bg-gray-50">
      <div className="max-w-7xl mx-auto">
        <div className="mb-6 flex items-center justify-between">
          <h1 className="text-3xl font-bold text-gray-900">軟體姿態 (Software Posture)</h1>
          <Link
            href="/"
            className="px-4 py-2 bg-gray-600 text-white rounded hover:bg-gray-700 transition-colors"
          >
            返回事件
          </Link>
        </div>

        <div className="mb-4 flex gap-4 items-center">
          <button
            onClick={fetchPostures}
            className="px-4 py-2 bg-blue-600 text-white rounded hover:bg-blue-700 transition-colors"
          >
            重新載入
          </button>
          <div className="text-sm text-gray-600">
            組件數量: <span className="font-semibold">{postures.length}</span>
          </div>
          <div className="text-sm text-gray-600">
            有漏洞: <span className="font-semibold text-red-600">
              {postures.filter(p => p.vulnCount > 0).length}
            </span>
          </div>
          <div className="text-sm text-gray-600">
            可更新: <span className="font-semibold text-blue-600">
              {postures.filter(p => p.updateAvailable).length}
            </span>
          </div>
        </div>

        {loading && <div className="text-center py-8 text-gray-600">載入中...</div>}
        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
            錯誤: {error}
          </div>
        )}

        {!loading && !error && (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {postures.length === 0 ? (
              <div className="col-span-full bg-white rounded-lg shadow p-8 text-center text-gray-500">
                尚無組件資料
              </div>
            ) : (
              postures.map((posture) => (
                <div key={posture.id} className="bg-white rounded-lg shadow p-6">
                  <div className="flex items-start justify-between mb-4">
                    <h3 className="text-lg font-semibold text-gray-900">{posture.component}</h3>
                    {posture.updateAvailable && (
                      <span className="px-2 py-1 text-xs font-semibold rounded bg-blue-100 text-blue-800">
                        可更新
                      </span>
                    )}
                  </div>

                  <div className="space-y-2 text-sm">
                    <div>
                      <span className="text-gray-600">當前版本:</span>
                      <span className="ml-2 font-mono text-gray-900">{posture.currentVersion}</span>
                    </div>

                    {posture.latestVersion && (
                      <div>
                        <span className="text-gray-600">最新版本:</span>
                        <span className="ml-2 font-mono text-gray-900">{posture.latestVersion}</span>
                      </div>
                    )}

                    <div>
                      <span className="text-gray-600">已知漏洞:</span>
                      <span className={`ml-2 px-2 py-1 text-xs font-semibold rounded ${getVulnColor(posture.vulnCount)}`}>
                        {posture.vulnCount}
                      </span>
                    </div>

                    {posture.imageDigest && (
                      <div>
                        <span className="text-gray-600">Image Digest:</span>
                        <span className="ml-2 font-mono text-xs text-gray-700">
                          {posture.imageDigest.substring(0, 16)}...
                        </span>
                      </div>
                    )}

                    {posture.lastScanTime && (
                      <div>
                        <span className="text-gray-600">最後掃描:</span>
                        <span className="ml-2 text-gray-700">{formatTime(posture.lastScanTime)}</span>
                      </div>
                    )}

                    {posture.lastUpdateTime && (
                      <div>
                        <span className="text-gray-600">最後更新:</span>
                        <span className="ml-2 text-gray-700">{formatTime(posture.lastUpdateTime)}</span>
                      </div>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>
        )}
      </div>
    </main>
  )
}

