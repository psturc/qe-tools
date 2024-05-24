module.exports = {
  extends: ['@commitlint/config-conventional'],
  ignores: [(message) => /^feat(deps): bump \[.+]\(.+\) from .+ to .+\.$/m.test(message)],
}
