local config = { settings = { gopls = {} } }
config.settings.gopls['local'] = 'github.com/dhemery/duffel'
vim.lsp.config('gopls', config)
