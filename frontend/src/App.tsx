import './App.css'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import LockerList from "./pages/LockerList/LockerList.tsx"
import Locker from "./pages/Locker/Locker.tsx"
import Admin from "./pages/Admin/Admin.tsx"

function App() {
  return (
    <>
      <Routes>
        <Route path="/" element={<LockerList />} />
        <Route path="/locker-:lockerId" element={<Locker />} />
        <Route path="/admin" element={<Admin />} />
      </Routes>
    </>
  )
}

export default App
