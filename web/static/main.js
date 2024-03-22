document.addEventListener("DOMContentLoaded", () => {
    document.body.addEventListener("htmx:afterRequest", () => {
        const finishLoadEvt = CustomEvent("custom:", { detail: { keepModalOpen: true } }) 
        document.body.dispatchEvent(finishLoadEvt)
    })
    document.body.addEventListener("htmx:beforeSwap", e => {
        let status = e.detail.xhr.status

        if (status < 400) {
            return
        }

        if (status >= 400 || status < 500) {
            e.detail.target = htmx.find("#modal-wrapper")
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


