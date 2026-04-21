module.exports = {
  plugins: [
    require("postcss-sorting")({
      order: ["custom-properties", "declarations", "rules", "at-rules"],
      "properties-order": [],
      "unspecified-properties-position": "bottomAlphabetical",
    }),
  ],
};
