const notyf = typeof Notyf !== 'undefined' ? new Notyf() : null
const nameEl = document.getElementById('name')
const phoneEl = document.getElementById('phonenumber')
const emailEl = document.getElementById('email')
const passEl = document.getElementById('password')
const modeToggle = document.getElementById('modeToggle')
const submitBtn = document.getElementById('submit')
const otpContainer = document.getElementById('otpform_container')
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
    } else {
        alert(msg)
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
    pending = {}
    updateUI()
})
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
                const res = await fetch(`/send_otp?email=${encodeURIComponent(email)}`)
                const j = await res.json().catch(() => ({}))
                showToast('OTP sent to email')
                pending = { name, ph, email, password }
                otpContainer && otpContainer.classList.remove('hidden')
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
                showToast('Signup successful')
                window.location.href = '/'
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
                window.location.href = '/'
            } else {
                showToast(j.error || 'Login failed', false)
            }
        } catch (err) {
            showToast('Login failed', false)
        }
    }
})
