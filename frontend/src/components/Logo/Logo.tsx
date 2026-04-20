import styles from "./Logo.module.css";

type LogoSize = "regular" | "compact";

type Props = {
  size?: LogoSize;
  className?: string;
};

function Logo({ size = "regular", className = "" }: Props) {
  const containerClassNames = [styles.container, styles[`container--${size}`], className]
    .filter(Boolean)
    .join(" ");

  const textClassNames = [styles.text, styles[`text--${size}`]].join(" ");

  return (
    <div className={containerClassNames}>
      <span className={textClassNames}>Lock'it</span>
    </div>
  );
}

export default Logo;
