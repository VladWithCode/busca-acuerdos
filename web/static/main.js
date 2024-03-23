document.addEventListener("DOMContentLoaded", () => {
    document.body.addEventListener("htmx:beforeRequest", () => {
        const startLoadEvt = new CustomEvent("custom:start-loading", { detail: { keepModalOpen: true } })
        document.body.dispatchEvent(startLoadEvt)
    })
    document.body.addEventListener("htmx:afterRequest", e => {
        if (e.detail.etc.stopFinishEvt) {
            return
        }

        const finishLoadEvt = new CustomEvent("custom:finish-loading", { detail: { keepModalOpen: true } }) 
        document.body.dispatchEvent(finishLoadEvt)
    })
    document.body.addEventListener("htmx:afterSwap", e => {
        if (e.detail.etc.animateConfirmModal === true) {
            const tl = gsap.timeline({ duration: 0.3, ease: "power2.inOut" })
            tl.pause()
            tl.fromTo("[data-confirm-modal]", { opacity: 0 }, { opacity: 1 }, "<")
            tl.fromTo("[data-confirm-modal-card]", { scale: 0 }, { scale: 1 }, "<")

            const finishLoadEvt = new CustomEvent("custom:finish-loading", { detail: { nextTween: tl, keepModalOpen: true } })    
            document.body.dispatchEvent(finishLoadEvt)
        }
    })
    document.body.addEventListener("htmx:beforeSwap", e => {
        let status = e.detail.xhr.status

        if (status < 400) {
            return
        }

        if (status >= 400 || status < 500) {
            let t = e.detail.xhr.getResponseHeader("Hx-Retarget")
            if (!t) {
                e.detail.etc.stopFinishEvt = true
                e.detail.target = htmx.find("#modal-wrapper")

                e.detail.etc.animateConfirmModal = true
            }
            e.detail.shouldSwap = true

            return
        }

        if (status >= 500) {
            if (typeof window.createErrorModal === 'function') {
                document
                    .querySelector()
                    .insertAdjacentHTML(
                        "beforeend",
                        window.createErrorModal({ message: "Ocurrio un error inesperado", btnLabel: "Aceptar" })
                    )

                return
            }

            alert("Ocurrió un error inesperado")
        }
    })
})


