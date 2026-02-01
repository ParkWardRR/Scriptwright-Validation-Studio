import { motion } from 'framer-motion'
import './Header.css'

export default function Header() {
  return (
    <motion.header
      className="header"
      initial={{ y: -20, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      transition={{ duration: 0.3 }}
    >
      <div className="header-content">
        <h1 className="header-title">
          <span className="header-icon">🧪</span>
          Userscript Validation Lab
        </h1>
        <p className="header-subtitle">Test userscripts with real browsers</p>
      </div>
    </motion.header>
  )
}
