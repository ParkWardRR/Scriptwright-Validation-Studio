import { useState } from 'react'
import { motion } from 'framer-motion'
import axios from 'axios'
import './RunForm.css'

interface RunFormProps {
  onRunCreated: () => void
}

export default function RunForm({ onRunCreated }: RunFormProps) {
  const [formData, setFormData] = useState({
    url: 'https://example.com',
    scriptSource: 'file',
    scriptPath: '',
    scriptURL: '',
    scriptGit: '',
    engine: 'tampermonkey',
    captureScreenshot: true,
    captureVideo: false,
    captureTrace: false,
  })
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState('')

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setIsSubmitting(true)
    setError('')

    try {
      const payload: any = {
        url: formData.url,
        engine: formData.engine,
        captureScreenshot: formData.captureScreenshot,
        captureVideo: formData.captureVideo,
        captureTrace: formData.captureTrace,
      }

      if (formData.scriptSource === 'file') {
        payload.scriptPath = formData.scriptPath
      } else if (formData.scriptSource === 'url') {
        payload.scriptURL = formData.scriptURL
      } else if (formData.scriptSource === 'git') {
        payload.scriptGit = formData.scriptGit
      }

      await axios.post('/v1/runs', payload)
      onRunCreated()

      // Reset form
      setFormData({
        ...formData,
        scriptPath: '',
        scriptURL: '',
        scriptGit: '',
      })
    } catch (err: any) {
      setError(err.response?.data?.error || 'Failed to create run')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <motion.form
      className="run-form"
      onSubmit={handleSubmit}
      initial={{ opacity: 0 }}
      animate={{ opacity: 1 }}
      transition={{ duration: 0.3 }}
    >
      <div className="form-group">
        <label>Target URL</label>
        <input
          type="url"
          value={formData.url}
          onChange={(e) => setFormData({ ...formData, url: e.target.value })}
          placeholder="https://example.com"
          required
        />
      </div>

      <div className="form-group">
        <label>Script Source</label>
        <div className="radio-group">
          <label className="radio-label">
            <input
              type="radio"
              value="file"
              checked={formData.scriptSource === 'file'}
              onChange={(e) => setFormData({ ...formData, scriptSource: e.target.value })}
            />
            File Path
          </label>
          <label className="radio-label">
            <input
              type="radio"
              value="url"
              checked={formData.scriptSource === 'url'}
              onChange={(e) => setFormData({ ...formData, scriptSource: e.target.value })}
            />
            URL
          </label>
          <label className="radio-label">
            <input
              type="radio"
              value="git"
              checked={formData.scriptSource === 'git'}
              onChange={(e) => setFormData({ ...formData, scriptSource: e.target.value })}
            />
            Git
          </label>
        </div>
      </div>

      {formData.scriptSource === 'file' && (
        <div className="form-group">
          <label>Script Path</label>
          <input
            type="text"
            value={formData.scriptPath}
            onChange={(e) => setFormData({ ...formData, scriptPath: e.target.value })}
            placeholder="/path/to/script.user.js"
            required
          />
        </div>
      )}

      {formData.scriptSource === 'url' && (
        <div className="form-group">
          <label>Script URL</label>
          <input
            type="url"
            value={formData.scriptURL}
            onChange={(e) => setFormData({ ...formData, scriptURL: e.target.value })}
            placeholder="https://example.com/script.user.js"
            required
          />
        </div>
      )}

      {formData.scriptSource === 'git' && (
        <div className="form-group">
          <label>Git Repository</label>
          <input
            type="text"
            value={formData.scriptGit}
            onChange={(e) => setFormData({ ...formData, scriptGit: e.target.value })}
            placeholder="https://github.com/user/repo"
            required
          />
        </div>
      )}

      <div className="form-group">
        <label>Engine</label>
        <select
          value={formData.engine}
          onChange={(e) => setFormData({ ...formData, engine: e.target.value })}
        >
          <option value="tampermonkey">Tampermonkey</option>
          <option value="violentmonkey">Violentmonkey</option>
          <option value="init-script">Init Script (no extension)</option>
        </select>
      </div>

      <div className="form-group">
        <label>Capture Options</label>
        <div className="checkbox-group">
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={formData.captureScreenshot}
              onChange={(e) => setFormData({ ...formData, captureScreenshot: e.target.checked })}
            />
            Screenshot
          </label>
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={formData.captureVideo}
              onChange={(e) => setFormData({ ...formData, captureVideo: e.target.checked })}
            />
            Video
          </label>
          <label className="checkbox-label">
            <input
              type="checkbox"
              checked={formData.captureTrace}
              onChange={(e) => setFormData({ ...formData, captureTrace: e.target.checked })}
            />
            Trace
          </label>
        </div>
      </div>

      {error && (
        <motion.div
          className="error-message"
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
        >
          {error}
        </motion.div>
      )}

      <button type="submit" className="submit-btn" disabled={isSubmitting}>
        {isSubmitting ? 'Creating Run...' : 'Start Test Run'}
      </button>
    </motion.form>
  )
}
