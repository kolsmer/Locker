import styles from './Logo.module.css';

function Logo () {
  return (
    <div className={styles.container}>
      <h1 className={styles.text}>Lock'it</h1>
    </div>
  )
}

export default Logo;