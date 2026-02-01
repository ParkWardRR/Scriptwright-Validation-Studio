import { useState } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { motion, AnimatePresence } from 'framer-motion'
import Dashboard from './pages/Dashboard'
import RunDetails from './pages/RunDetails'
import Header from './components/Header'
import './App.css'

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

type Page = 'dashboard' | 'run-details'

function App() {
  const [currentPage, setCurrentPage] = useState<Page>('dashboard')
  const [selectedRunId, setSelectedRunId] = useState<string>('')

  const viewRun = (runId: string) => {
    setSelectedRunId(runId)
    setCurrentPage('run-details')
  }

  const goBack = () => {
    setCurrentPage('dashboard')
    setSelectedRunId('')
  }

  return (
    <QueryClientProvider client={queryClient}>
      <div className="app">
        <Header />
        <AnimatePresence mode="wait">
          {currentPage === 'dashboard' && (
            <motion.div
              key="dashboard"
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -20 }}
              transition={{ duration: 0.2 }}
            >
              <Dashboard onViewRun={viewRun} />
            </motion.div>
          )}
          {currentPage === 'run-details' && (
            <motion.div
              key="run-details"
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: 20 }}
              transition={{ duration: 0.2 }}
            >
              <RunDetails runId={selectedRunId} onBack={goBack} />
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </QueryClientProvider>
  )
}

export default App
