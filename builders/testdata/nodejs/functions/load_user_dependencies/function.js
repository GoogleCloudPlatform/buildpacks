/**
 * Renders a template.
 * This function tests if the Functions Framework can load the "ejs" package.
 *
 * @param {!Object} req request context.
 * @param {!Object} res response context.
 */
function testFunction(req, res) {
  res.render("index.ejs");
}

module.exports = {
  testFunction,
};
