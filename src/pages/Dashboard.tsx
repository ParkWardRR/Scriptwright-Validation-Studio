import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { motion } from 'framer-motion'
import axios from 'axios'
import RunForm from '../components/RunForm'
import RunCard from '../components/RunCard'
import './Dashboard.css'

interface Run {
  id: string
  status: 'running' | 'completed' | 'failed'
  url: string
  engine: string
  timestamp: string
  duration?: number
}

interface DashboardProps {
  onViewRun: (runId: string) => void
}

export default function Dashboard({ onViewRun }: DashboardProps) {
  const [isFormVisible, setIsFormVisible] = useState(true)

  const { data: runs = [], isLoading, refetch } = useQuery<Run[]>({
    queryKey: ['runs'],
    queryFn: async () => {
      const response = await axios.get('/v1/runs')
      return response.data
    },
    refetchInterval: 5000, // Poll every 5 seconds
  })

  const handleRunCreated = () => {
    refetch()
    setIsFormVisible(false)
  }

  return (
    <div className="dashboard">
      <div className="dashboard-content">
        <motion.div
          className="dashboard-section"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
        >
          <div className="section-header">
            <h2>New Test Run</h2>
            <button
              className="toggle-btn"
              onClick={() => setIsFormVisible(!isFormVisible)}
            >
              {isFormVisible ? 'Hide' : 'Show'}
            </button>
          </div>
          {isFormVisible && <RunForm onRunCreated={handleRunCreated} />}
        </motion.div>

        <motion.div
          className="dashboard-section"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
        >
          <h2>Recent Runs</h2>
          {isLoading ? (
            <div className="loading">Loading runs...</div>
          ) : runs.length === 0 ? (
            <div className="empty-state">
              <p>No runs yet. Create your first test run above.</p>
            </div>
          ) : (
            <div className="runs-grid">
              {runs.map((run, index) => (
                <RunCard
                  key={run.id}
                  run={run}
                  index={index}
                  onView={() => onViewRun(run.id)}
                />
              ))}
            </div>
          )}
        </motion.div>
      </div>
    </div>
  )
}
