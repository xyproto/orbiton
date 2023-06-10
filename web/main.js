document.addEventListener("DOMContentLoaded", function() {
    const screenshots = document.querySelectorAll(".screenshot");
    const modal = document.getElementById("modal");
    const modalImg = document.getElementById("modal-img");
    const modalCaption = document.getElementById("modal-caption");
    const closeModal = document.getElementById("close-modal");

    screenshots.forEach((screenshot, index) => {
        screenshot.addEventListener("click", function() {
            modal.style.display = "block";
            modalImg.src = this.src;
            modalCaption.innerHTML = this.nextElementSibling.textContent;
        });
    });

    closeModal.addEventListener("click", function() {
        modal.style.display = "none";
    });

    modalImg.addEventListener("click", function() {
        modal.style.display = "none";
    });
});
