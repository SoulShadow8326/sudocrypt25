const notyf = typeof Notyf !== 'undefined' ? new Notyf() : null
const nameEl = document.getElementById('name')
const phoneEl = document.getElementById('phonenumber')
const emailEl = document.getElementById('email')
const passEl = document.getElementById('password')
const modeToggle = document.getElementById('modeToggle')
const submitBtn = document.getElementById('submit')
const otpContainer = document.getElementById('otpform_container')
const inputListEl = document.getElementById('inputList')
const authTxt = document.getElementById('authTxt')
const otpInputs = () => Array.from(document.querySelectorAll('.otp-input'))
let mode = 'signup'
let pending = {}
function readOtp() {
    return otpInputs().map(i => i.value || '').join('')
}
function showToast(msg, ok = true) {
    if (notyf) {
        ok ? notyf.success(msg) : notyf.error(msg)
    }
}
function updateUI() {
    if (mode === 'signup') {
        modeToggle && (modeToggle.textContent = 'Already Registered?')
        submitBtn && (submitBtn.textContent = 'Register')
        authTxt && (authTxt.textContent = 'Register')
        nameEl && (nameEl.style.display = '')
        phoneEl && (phoneEl.style.display = '')
    } else {
        modeToggle && (modeToggle.textContent = 'No Existing Account?')
        submitBtn && (submitBtn.textContent = 'Login')
        authTxt && (authTxt.textContent = 'Login')
        nameEl && (nameEl.style.display = 'none')
        phoneEl && (phoneEl.style.display = 'none')
    }
}

updateUI()

modeToggle && modeToggle.addEventListener('click', (e) => {
    e && e.preventDefault && e.preventDefault()
    mode = mode === 'signup' ? 'login' : 'signup'
    if (otpContainer) {
        otpContainer.classList.add('hidden')
    }
    if (inputListEl) {
        inputListEl.classList.remove('hidden')
    }
    pending = {}
    updateUI()
})

function setupOtpInputsBehavior() {
    const inputs = otpInputs()
    if (!inputs || inputs.length === 0) return
    inputs.forEach((input, idx) => {
        if (input.dataset.otpBound) return
        input.dataset.otpBound = '1'
        input.setAttribute('inputmode', 'numeric')
        input.addEventListener('input', (e) => {
            let v = e.target.value || ''
            v = v.replace(/\D/g, '')
            if (v.length > 1) {
                const chars = v.split('')
                for (let i = 0; i < chars.length && (idx + i) < inputs.length; i++) {
                    inputs[idx + i].value = chars[i]
                }
                const next = Math.min(idx + chars.length, inputs.length - 1)
                inputs[next].focus()
            } else {
                e.target.value = v
                if (v !== '' && idx < inputs.length - 1) {
                    inputs[idx + 1].focus()
                }
            }
        })
        input.addEventListener('paste', (e) => {
            e.preventDefault()
            const paste = (e.clipboardData || window.clipboardData).getData('text') || ''
            const digits = paste.replace(/\D/g, '').slice(0, inputs.length - idx)
            for (let i = 0; i < digits.length && (idx + i) < inputs.length; i++) {
                inputs[idx + i].value = digits[i]
            }
            const next = Math.min(idx + digits.length, inputs.length - 1)
            inputs[next].focus()
        })
        input.addEventListener('keydown', (e) => {
            if (e.key === 'Backspace' && input.value === '' && idx > 0) {
                inputs[idx - 1].focus()
                inputs[idx - 1].value = ''
                e.preventDefault()
            }
        })
    })
}

setupOtpInputsBehavior()
submitBtn && submitBtn.addEventListener('click', async (e) => {
    e.preventDefault()
    const email = emailEl && emailEl.value.trim()
    const password = passEl && passEl.value
    if (mode === 'signup') {
        const name = nameEl && nameEl.value.trim()
        const ph = phoneEl && phoneEl.value.trim()
        if (!email || !password || !name || !ph) {
            showToast('Missing fields', false)
            return
        }
        if (!otpContainer || otpContainer.classList.contains('hidden')) {
            try {
                const res = await fetch('/send_otp', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ name, phonenumber: ph, email, password })
                })
                const j = await res.json().catch(() => ({}))
                if (res.ok) {
                    showToast('OTP sent to email')
                    pending = { name, ph, email, password }
                    otpContainer && otpContainer.classList.remove('hidden')
                    if (inputListEl) {
                        inputListEl.classList.add('hidden')
                    }
                    const firstOtp = document.querySelector('.otp-input')
                    if (firstOtp) {
                        setTimeout(() => firstOtp.focus(), 100)
                    }
                } else {
                    showToast(j.error || 'Failed to send OTP', false)
                }
            } catch (err) {
                showToast('Failed to send OTP', false)
            }
            return
        }
        const otp = readOtp()
        if (otp.length !== 6) {
            showToast('Enter 6 digit OTP', false)
            return
        }
        const params = new URLSearchParams({ method: 'signup', email: pending.email, password: pending.password, name: pending.name, phonenumber: pending.ph, otp })
        try {
            const res = await fetch('/api/auth?' + params.toString())
            const j = await res.json().catch(() => ({}))
                if (res.ok) {
                    showToast('Auth successful')
                    const params = new URLSearchParams(window.location.search || '')
                    const from = params.get('from') || '/play'
                    const sep = from.includes('?') ? '&' : '?'
                    window.location.href = from + sep + 'auth=signup'
                } else {
                showToast(j.error || 'Signup failed', false)
            }
        } catch (err) {
            showToast('Signup failed', false)
        }
        return
    }
    if (mode === 'login') {
        if (!email || !password) {
            showToast('Missing fields', false)
            return
        }
        const params = new URLSearchParams({ method: 'login', email, password })
        try {
            const res = await fetch('/api/auth?' + params.toString())
            const j = await res.json().catch(() => ({}))
            if (res.ok) {
                showToast('Login successful')
                const params = new URLSearchParams(window.location.search || '')
                const from = params.get('from') || '/play'
                const sep = from.includes('?') ? '&' : '?'
                window.location.href = from + sep + 'auth=login'
            } else {
                showToast(j.error || 'Login failed', false)
            }
        } catch (err) {
            showToast('Login failed', false)
        }
    }
})

function handleRedirectToast() {
    try {
        const params = new URLSearchParams(window.location.search || '')
        const from = params.get('from') || ''
        if (params.get('toast') === '1' && (from === '/play' || from === '/leaderboard')) {
            showToast('You must be logged in to access this page. Please login or sign up to continue.')
            const cleanUrl = window.location.pathname + window.location.hash
            if (history && history.replaceState) {
                history.replaceState(null, '', cleanUrl)
            }
        }
        const authType = params.get('auth')
        if (authType === 'login' || authType === 'signup') {
            if (window.location.pathname === '/play') {
                if (authType === 'login') showToast('Login successful')
                if (authType === 'signup') showToast('Auth successful')
                const cleanUrl = window.location.pathname + window.location.hash
                if (history && history.replaceState) {
                    history.replaceState(null, '', cleanUrl)
                }
            }
        }
    } catch (err) {
    }
}

handleRedirectToast()
