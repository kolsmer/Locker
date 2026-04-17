module.exports = {
  plugins: [
    require("postcss-sorting")({
      "properties-order": "alphabetical-order",
    }),
  ],
};
