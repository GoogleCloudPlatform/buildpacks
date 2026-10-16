document.addEventListener("DOMContentLoaded", () => {
    console.log("SPA Client-side Scripts Executing Perfectly!");
    const status = document.createElement("p");
    status.style.color = "green";
    status.style.fontWeight = "bold";
    status.textContent = "✓ Executed accompanying JS asset.";
    document.body.appendChild(status);
});
