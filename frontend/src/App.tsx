import './App.css'
import { Navigate, Route, Routes, useParams } from 'react-router-dom'
import LockerList from "./pages/LockerList/LockerList.tsx"
import Locker from "./pages/Locker/Locker.tsx"
import Admin from "./pages/Admin/Admin.tsx"

function LegacyLockerRedirect() {
  const { lockerSlug } = useParams<{ lockerSlug: string }>()

  if (!lockerSlug || !lockerSlug.startsWith('locker-')) {
    return <Navigate to="/" replace />
  }

  const lockerId = lockerSlug.slice('locker-'.length)

  if (!lockerId) {
    return <Navigate to="/" replace />
  }

  return <Navigate to={`/locker/${lockerId}`} replace />
}

function App() {
  return (
    <>
      <Routes>
        <Route path="/" element={<LockerList />} />
        <Route path="/locker/:lockerId" element={<Locker />} />
        <Route path="/admin" element={<Admin />} />
        <Route path="/:lockerSlug" element={<LegacyLockerRedirect />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </>
  )
}

export default App
