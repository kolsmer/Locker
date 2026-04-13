import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './App.tsx'
import Logo from './components/Logo/Logo.tsx'
import Button from './components/Button/Button.tsx'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <App />
      <Logo></Logo>
      <Button variant="regular" fullWidth={false}>L</Button>
    </BrowserRouter>
  </StrictMode>,
)
