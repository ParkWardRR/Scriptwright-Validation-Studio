import { motion } from 'framer-motion'
import './RunCard.css'

interface Run {
  id: string
  status: 'running' | 'completed' | 'failed'
  url: string
  engine: string
  timestamp: string
  duration?: number
}

interface RunCardProps {
  run: Run
  index: number
  onView: () => void
}

export default function RunCard({ run, index, onView }: RunCardProps) {
  const statusColors = {
    running: 'var(--color-primary)',
    completed: 'var(--color-success)',
    failed: 'var(--color-error)',
  }

  const statusIcons = {
    running: '⏳',
    completed: '✅',
    failed: '❌',
  }

  const formatDuration = (ms?: number) => {
    if (!ms) return '—'
    const seconds = Math.floor(ms / 1000)
    if (seconds < 60) return `${seconds}s`
    const minutes = Math.floor(seconds / 60)
    const remainingSeconds = seconds % 60
    return `${minutes}m ${remainingSeconds}s`
  }

  const formatTime = (timestamp: string) => {
    try {
      const date = new Date(timestamp)
      return date.toLocaleString()
    } catch {
      return timestamp
    }
  }

  return (
    <motion.div
      className="run-card"
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.05, duration: 0.3 }}
      whileHover={{ y: -4, boxShadow: 'var(--shadow)' }}
      onClick={onView}
    >
      <div className="run-card-header">
        <div className="run-status" style={{ color: statusColors[run.status] }}>
          <span className="status-icon">{statusIcons[run.status]}</span>
          <span className="status-text">{run.status}</span>
        </div>
        <div className="run-id">{run.id.slice(0, 8)}</div>
      </div>

      <div className="run-card-body">
        <div className="run-url" title={run.url}>
          {run.url}
        </div>
        <div className="run-meta">
          <div className="meta-item">
            <span className="meta-label">Engine:</span>
            <span className="meta-value">{run.engine}</span>
          </div>
          <div className="meta-item">
            <span className="meta-label">Duration:</span>
            <span className="meta-value">{formatDuration(run.duration)}</span>
          </div>
        </div>
      </div>

      <div className="run-card-footer">
        <div className="run-time">{formatTime(run.timestamp)}</div>
        <button className="view-btn">View Details →</button>
      </div>
    </motion.div>
  )
}
