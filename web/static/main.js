document.addEventListener("DOMContentLoaded", () => {
    document.body.addEventListener("htmx:responseError", e => {
        const message = e.detail.xhr.responseText;

        alert(message || "Ocurrio un error al recuperar el expediente solicitado.")
    })
})


