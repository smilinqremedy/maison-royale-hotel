document.addEventListener("DOMContentLoaded", function () {
    const checkInInput = document.getElementById("check-in");
    const checkOutInput = document.getElementById("check-out");
    const roomSelect = document.getElementById("room");
    const bookingForm = document.querySelector(".booking-form");

    if (!checkInInput || !checkOutInput || !roomSelect || !bookingForm) {
        return;
    }

    // Room prices per night
    const roomPrices = {
        "1": 85000,
        "2": 140000,
        "3": 250000
    };

    // Create price summary
    const priceSummary = document.createElement("div");

    priceSummary.className = "price-summary";

    priceSummary.innerHTML = `
        <div class="price-row">
            <span>Price per night</span>
            <strong id="price-per-night">₦0</strong>
        </div>

        <div class="price-row">
            <span>Number of nights</span>
            <strong id="number-of-nights">0</strong>
        </div>

        <div class="price-total">
            <span>Estimated total</span>
            <strong id="estimated-total">₦0</strong>
        </div>
    `;

    bookingForm.appendChild(priceSummary);

    const pricePerNightElement =
        document.getElementById("price-per-night");

    const numberOfNightsElement =
        document.getElementById("number-of-nights");

    const estimatedTotalElement =
        document.getElementById("estimated-total");


    function formatCurrency(amount) {
        return new Intl.NumberFormat("en-NG", {
            style: "currency",
            currency: "NGN",
            maximumFractionDigits: 0
        }).format(amount);
    }


    function calculatePrice() {
        const checkInValue = checkInInput.value;
        const checkOutValue = checkOutInput.value;
        const roomID = roomSelect.value;

        const pricePerNight = roomPrices[roomID] || 0;

        pricePerNightElement.textContent =
            formatCurrency(pricePerNight);

        if (!checkInValue || !checkOutValue) {
            numberOfNightsElement.textContent = "0";
            estimatedTotalElement.textContent = formatCurrency(0);
            return;
        }

        const checkInDate = new Date(checkInValue + "T00:00:00");
        const checkOutDate = new Date(checkOutValue + "T00:00:00");

        const difference =
            checkOutDate.getTime() - checkInDate.getTime();

        const millisecondsPerDay =
            1000 * 60 * 60 * 24;

        const numberOfNights =
            Math.round(difference / millisecondsPerDay);


        if (numberOfNights <= 0) {
            numberOfNightsElement.textContent = "0";
            estimatedTotalElement.textContent = formatCurrency(0);
            return;
        }

        const estimatedTotal =
            pricePerNight * numberOfNights;

        numberOfNightsElement.textContent =
            numberOfNights;

        estimatedTotalElement.textContent =
            formatCurrency(estimatedTotal);
    }


    checkInInput.addEventListener(
        "change",
        calculatePrice
    );

    checkOutInput.addEventListener(
        "change",
        calculatePrice
    );

    roomSelect.addEventListener(
        "change",
        calculatePrice
    );


    // Prevent selecting a checkout date
    // before or equal to the check-in date.
    checkInInput.addEventListener("change", function () {

        if (!checkInInput.value) {
            return;
        }

        checkOutInput.min = checkInInput.value;

        if (
            checkOutInput.value &&
            checkOutInput.value <= checkInInput.value
        ) {
            checkOutInput.value = "";
        }

        calculatePrice();
    });


    // Validate dates before submitting
    bookingForm.addEventListener("submit", function (event) {

        const checkInValue = checkInInput.value;
        const checkOutValue = checkOutInput.value;

        if (!checkInValue || !checkOutValue) {
            return;
        }

        const checkInDate =
            new Date(checkInValue + "T00:00:00");

        const checkOutDate =
            new Date(checkOutValue + "T00:00:00");


        if (checkOutDate <= checkInDate) {

            event.preventDefault();

            alert(
                "Check-out date must be after the check-in date."
            );

            return;
        }

    });


    // Calculate initial price
    calculatePrice();
});