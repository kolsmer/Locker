import type { InputHTMLAttributes } from "react";
import styles from "./Input.module.css";

type InputVariant = "big" | "regular" | "compact";

type Props = {
	variant?: InputVariant;
	fullWidth?: boolean;
} & InputHTMLAttributes<HTMLInputElement>;

function Input({
	variant = "regular",
	fullWidth = false,
	className = "",
	...rest
}: Props) {
	const classes = [
		styles.input,
		styles[`input--${variant}`],
		fullWidth ? styles["input--full"] : "",
		className,
	]
		.filter(Boolean)
		.join(" ");

	return <input className={classes} {...rest} />;
}

export default Input;
