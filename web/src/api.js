// Shared HTTP API helper — automatically attaches auth token.
export async function api(method, path, body) {
  const opts = { method, headers: {} }
  const token = localStorage.getItem('singa_token')
  if (token && token !== 'noauth') {
    opts.headers['X-Auth-Token'] = token
  }
  if (body) {
    opts.headers['Content-Type'] = 'application/json'
    opts.body = JSON.stringify(body)
  }
  const res = await fetch('/api' + path, opts)
  const data = await res.json()
  if (res.status === 401) {
    // Token expired or invalid
    localStorage.removeItem('singa_token')
    window.location.hash = '#/login'
    throw new Error('请重新登录')
  }
  if (!res.ok) throw new Error(data.error || res.statusText)
  return data
}
