import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import axios from 'axios'
import './RunDetails.css'

interface RunDetailsProps {
  runId: string
  onBack: () => void
}

interface RunResult {
  id: string
  status: 'running' | 'completed' | 'failed'
  url: string
  engine: string
  timestamp: string
  duration?: number
  screenshot?: string
  video?: string
  trace?: string
  logs?: Array<{ timestamp: string; level: string; message: string }>
  assertions?: Array<{ type: string; passed: boolean; message: string }>
  visualDiff?: {
    baseline?: string
    actual?: string
    diff?: string
    pixelsDifferent?: number
    percentDifferent?: number
  }
  error?: string
}

export default function RunDetails({ runId, onBack }: RunDetailsProps) {
  const { data: run, isLoading } = useQuery<RunResult>({
    queryKey: ['run', runId],
    queryFn: async () => {
      const response = await axios.get(`/v1/runs/${runId}`)
      return response.data
    },
    refetchInterval: (query) => {
      const data = query.state.data
      return data?.status === 'running' ? 2000 : false
    },
  })

  if (isLoading) {
    return (
      <div className="run-details">
        <div className="loading">Loading run details...</div>
      </div>
    )
  }

  if (!run) {
    return (
      <div className="run-details">
        <div className="error">Run not found</div>
      </div>
    )
  }

  const statusColors = {
    running: 'var(--color-primary)',
    completed: 'var(--color-success)',
    failed: 'var(--color-error)',
  }

  return (
    <motion.div
      className="run-details"
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.3 }}
    >
      <div className="details-header">
        <button className="back-btn" onClick={onBack}>
          ← Back to Dashboard
        </button>
        <div className="details-title">
          <h1>Run Details</h1>
          <div className="run-status" style={{ color: statusColors[run.status] }}>
            {run.status}
          </div>
        </div>
      </div>

      <div className="details-content">
        <motion.section
          className="details-section"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
        >
          <h2>Overview</h2>
          <div className="overview-grid">
            <div className="overview-item">
              <span className="overview-label">Run ID:</span>
              <span className="overview-value">{run.id}</span>
            </div>
            <div className="overview-item">
              <span className="overview-label">URL:</span>
              <a href={run.url} target="_blank" rel="noopener noreferrer" className="overview-link">
                {run.url}
              </a>
            </div>
            <div className="overview-item">
              <span className="overview-label">Engine:</span>
              <span className="overview-value">{run.engine}</span>
            </div>
            <div className="overview-item">
              <span className="overview-label">Timestamp:</span>
              <span className="overview-value">{new Date(run.timestamp).toLocaleString()}</span>
            </div>
            {run.duration && (
              <div className="overview-item">
                <span className="overview-label">Duration:</span>
                <span className="overview-value">{(run.duration / 1000).toFixed(2)}s</span>
              </div>
            )}
          </div>
        </motion.section>

        {run.error && (
          <motion.section
            className="details-section error-section"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
          >
            <h2>Error</h2>
            <pre className="error-output">{run.error}</pre>
          </motion.section>
        )}

        {run.screenshot && (
          <motion.section
            className="details-section"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
          >
            <h2>Screenshot</h2>
            <img src={`/runs/${runId}/${run.screenshot}`} alt="Screenshot" className="artifact-img" />
            <a href={`/runs/${runId}/${run.screenshot}`} download className="download-btn">
              Download Screenshot
            </a>
          </motion.section>
        )}

        {run.video && (
          <motion.section
            className="details-section"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.3 }}
          >
            <h2>Video</h2>
            <video controls className="artifact-video" src={`/runs/${runId}/${run.video}`} />
            <a href={`/runs/${runId}/${run.video}`} download className="download-btn">
              Download Video
            </a>
          </motion.section>
        )}

        {run.trace && (
          <motion.section
            className="details-section"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.4 }}
          >
            <h2>Trace</h2>
            <a href={`/runs/${runId}/${run.trace}`} download className="download-btn">
              Download Trace (Playwright Trace Viewer)
            </a>
          </motion.section>
        )}

        {run.visualDiff && (
          <motion.section
            className="details-section"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.5 }}
          >
            <h2>Visual Diff</h2>
            <div className="diff-stats">
              <div className="diff-stat">
                <span className="diff-label">Pixels Different:</span>
                <span className="diff-value">{run.visualDiff.pixelsDifferent?.toLocaleString()}</span>
              </div>
              <div className="diff-stat">
                <span className="diff-label">Percent Different:</span>
                <span className="diff-value">{run.visualDiff.percentDifferent?.toFixed(2)}%</span>
              </div>
            </div>
            <div className="diff-images">
              {run.visualDiff.baseline && (
                <div className="diff-image-container">
                  <h3>Baseline</h3>
                  <img src={`/runs/${runId}/${run.visualDiff.baseline}`} alt="Baseline" />
                </div>
              )}
              {run.visualDiff.actual && (
                <div className="diff-image-container">
                  <h3>Actual</h3>
                  <img src={`/runs/${runId}/${run.visualDiff.actual}`} alt="Actual" />
                </div>
              )}
              {run.visualDiff.diff && (
                <div className="diff-image-container">
                  <h3>Difference</h3>
                  <img src={`/runs/${runId}/${run.visualDiff.diff}`} alt="Diff" />
                </div>
              )}
            </div>
          </motion.section>
        )}

        {run.assertions && run.assertions.length > 0 && (
          <motion.section
            className="details-section"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.6 }}
          >
            <h2>Assertions</h2>
            <div className="assertions-list">
              {run.assertions.map((assertion, index) => (
                <div
                  key={index}
                  className={`assertion-item ${assertion.passed ? 'passed' : 'failed'}`}
                >
                  <span className="assertion-icon">{assertion.passed ? '✅' : '❌'}</span>
                  <span className="assertion-type">{assertion.type}</span>
                  <span className="assertion-message">{assertion.message}</span>
                </div>
              ))}
            </div>
          </motion.section>
        )}

        {run.logs && run.logs.length > 0 && (
          <motion.section
            className="details-section"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.7 }}
          >
            <h2>Logs</h2>
            <div className="logs-container">
              {run.logs.map((log, index) => (
                <div key={index} className={`log-line log-${log.level}`}>
                  <span className="log-time">{log.timestamp}</span>
                  <span className="log-level">{log.level.toUpperCase()}</span>
                  <span className="log-message">{log.message}</span>
                </div>
              ))}
            </div>
          </motion.section>
        )}
      </div>
    </motion.div>
  )
}
