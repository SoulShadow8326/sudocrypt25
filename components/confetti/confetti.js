const canvas = document.getElementById('canvas')
if (canvas) {
    const ctx = canvas.getContext('2d')
    function resize() {
        canvas.width = canvas.clientWidth
        canvas.height = canvas.clientHeight
    }
    window.addEventListener('resize', resize)
    resize()
}
