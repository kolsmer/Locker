import { useParams } from "react-router-dom"

function Locker() {
  const { lockerId } = useParams<{ lockerId: string}>();
  return (
    <div>
      <h1>Locker {lockerId}</h1>
    </div>
  )
}

export default Locker;