import type { ButtonHTMLAttributes, ReactNode } from "react"
import styles from "./Button.module.css";

type ButtonVariant = "big" | "regular";

type Props = {
  children: ReactNode;
  variant?: ButtonVariant;
  fullWidth?: boolean;
} & ButtonHTMLAttributes<HTMLButtonElement>;

function Button ({ children, variant = "regular", fullWidth = false, className = "", ...rest }: Props) {

const classes = [
  styles.btn,
  styles[`btn--${variant}`],
  fullWidth ? styles["btn--full"] : "",
  className,
]
  .filter(Boolean)
  .join(" ");

  return (
    <button className={classes} {...rest}>
      {children}
    </button>
  );
}

export default Button;