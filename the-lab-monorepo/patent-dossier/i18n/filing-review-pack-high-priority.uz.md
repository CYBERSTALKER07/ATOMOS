# Pegasus: Yuqori Ustuvor Patent Taqdimoti Uchun Review Paket

**Maqsad**
Bu paket Pegasus patent mazmuni eng ravshan ko'rinadigan ekranlarning o'zbekcha sharhidir. Overlay JSON fayllarida katalog ko'proq. Bu yerda esa izoh ko'proq: ekran nima qiladi, state zanjirida nimani ushlab turadi va nega bu patent uchun muhim.

Tanlangan yuzalar to'liq zanjirni yopadi: rol bo'yicha kirish, tijoriy niyatni ushlash, AI talab signali, settlement konfiguratsiyasi, ombor dispatch logikasi va delivery proof yoki correction oqimlari.

**Arxitektura Tezisi**
Bu yuzalar Pegasus bir-biridan uzilgan ilovalar to'plami emasligini ko'rsatadi. Bir xil tijoriy haqiqat onboarding, talab prognozi, checkout, settlement, dispatch, proof va correction orqali yuradi va har bosqichda boshqariladigan holatda qoladi.

**1. Ta'minotchi Ro'yxatdan O'tishi**
Asosiy dossye: patent-dossier/page-dossiers/web-supplier-register.json

Ta'minotchi registratsiyasi to'rt bosqichli wizard ko'rinishida qurilgan bo'lib, har bir bosqich keyingi operatsion xulq uchun alohida semantik qatlam yaratadi. Avval account identity, keyin warehouse location, undan so'ng business va fleet profile, category selection va yakunda payment gateway preference olinadi. Shu sababli bu sahifa oddiy signup emas; u future supply node'ni geometriya, category scope, cold-chain flag va settlement tanlovi bilan birga shakllantiradi.

Step indicator yuqorida joylashgani, markazda esa yagona wizard-card turishi foydalanuvchini qat'iy izchillik bo'yicha olib yuradi. Step 2 dagi Locate tugmasi brauzer geolocation va reverse geocode'ni operatsion warehouse anchor'ga aylantiradi. Patent nuqtai nazaridan bu yuzada identity, location, category va payment preference bitta onboarding contract ichida birlashadi.

**2. Ta'minotchi Dashboard Va AI Future Demand**
Asosiy dossye: patent-dossier/page-dossiers/web-supplier-dashboard.json

Supplier dashboard himoyalangan shell ichida birinchi navbatda analytics va demand intelligence'ni ko'rsatadi. Yuqorida sarlavha va Dispatch Control Room havolasi bor, undan keyin AI Future Demand kartasi, KPI grid, SKU velocity chart va volume share jadvali joylashadi. Demak foydalanuvchi faqat o'tgan savdolarni emas, balki kutilayotgan retail talabni ham bitta ishchi yuzada ko'radi.

Patent uchun bu ekran muhim, chunki AI forecast alohida report sifatida emas, supplierning amaliy qaror qabul qilish qobig'ida paydo bo'ladi. Future-demand kartasi dispatch va analytics drill-down bilan yonma-yon turadi, shuning uchun mashina tomonidan yaratilgan signal darhol operatsion qarorga aylanishi mumkin. Shu ekran claim family bo'yicha forecast-to-procurement bog'lanishini ko'rsatadi.

**3. Ta'minotchi Mahsulot Portfeli Va Aktivlik Boshqaruvi**
Asosiy dossye: patent-dossier/page-dossiers/web-supplier-products.json

Mahsulotlar sahifasi SKU boshqaruvining markaziy ish joyi sifatida ishlaydi. Yuqorida My Products sarlavhasi, ro'yxatga olingan SKU soni va Add Product havolasi bor. Uning ostida jami SKU, aktiv, noaktiv va katalog qiymatini ko'rsatadigan KPI strip joylashgan. Keyin search qatori, refresh tugmasi, category chip'lar va oxirida product card'lar grid'i keladi.

Patent mazmuni uchun asosiy ahamiyat card darajasidagi Activate va Deactivate nazoratlaridadir. Har bir mahsulot card'i status badge, category pill, narx, SKU va aktivlik icon tugmasini birlashtiradi. Natijada ta'minotchi alohida sahifaga o'tmasdan turib mahsulotni tijoriy oqimga qo'shishi yoki undan chiqarishi mumkin. Bu yuzada supplier governance bilan retailerga ko'rinadigan commercial availability o'rtasidagi bog'liqlik aniq ko'rinadi.

**4. Ta'minotchi Payment Gateway Konfiguratsiyasi**
Asosiy dossye: patent-dossier/page-dossiers/web-supplier-payment-config.json

Payment Gateways sahifasi settlement mesh supplier-scoped credential boshqaruviga tayanishini ochib beradi. Har bir provayder alohida card ko'rinishida berilgan: Click, Payme va Global Pay uchun icon, status, merchant preview va Connect, Manual setup, Update yoki Deactivate kabi amallar mavjud. Manual form kerak bo'lganda aynan shu provider card ichida kengayadi.

Huquqiy jihatdan muhim nuqta shundaki, maxfiy credential'lar umumiy panel sifatida emas, aynan tanlangan gateway kontekstida boshqariladi. Secret key ochiq ko'rsatilmaydi, merchant va service maydonlari esa provayderga xos yordamchi matn bilan birga keladi. Shunday qilib, bu sahifa unified checkout va webhook reconciliation bilan bog'lanadigan supplier-specific settlement rail'larni aniq isbotlaydi.

**5. Retailer Android: Mahsulot Detali Va Savatga Qo'shish Niyati**
Asosiy dossye: patent-dossier/page-dossiers/retailer-android-secondary-surfaces.json
Yuza: android-retailer-product-detail

Android product detail sahifasi product ko'rishni aniq tijoriy niyatga aylantiradi. Hero image, info section, variant selector, quantity stepper, metadata maydoni va fixed Add to Cart bar birgalikda ishlaydi. Retailer shu yerning o'zida variant tanlaydi, sonni o'zgartiradi, kerak bo'lsa auto-order holatini yoqadi va savatga qo'shadi.

Bu yuzaning patent qiymati ikki qatlamdan iborat. Birinchisi, commercial intent granular tarzda variant plus quantity plus add-to-cart ko'rinishida yoziladi. Ikkinchisi, auto-order toggle sababli shu bir mahsulot ham tezkor xarid obyekti, ham kelajakdagi replenishment policy elementi bo'lib qoladi. Demak sahifa bir martalik savdo va davomli avtomatik procurement o'rtasidagi ko'prik vazifasini bajaradi.

**6. Retailer iOS: Savat**
Asosiy dossye: patent-dossier/page-dossiers/retailer-ios-cart.json

iOS savat ekrani unified checkout oldidan barcha item niyatlarini jamlaydigan qatlamdir. Cart count, Clear All, har bir mahsulot card'i, quantity stepper va delete affordance birgalikda ishlaydi; pastda esa subtotal, delivery va total ko'rsatilgan sticky summary bar mavjud. Savat bo'sh bo'lsa, Browse Catalog CTA bilan alohida empty state ko'rsatiladi.

Patent nuqtai nazaridan savat faqat vaqtinchalik ro'yxat emas. U quantity ni o'zgartirish, item ni o'chirish va yakuniy summani qayta hisoblash orqali keyingi checkout uchun deterministik order structure yaratadi. Shu sababli bu ekran basket-level computation surface sifatida ko'riladi va unified order intent shakllanishining bevosita boshlanish nuqtasidir.

**7. Retailer iOS: Checkout Va Buyurtmani Yakunlash**
Asosiy dossye: patent-dossier/page-dossiers/retailer-ios-checkout.json

Checkout sahifasi retail commerce oqimining yakuniy darvozasidir. Yuqorida dismiss tugmasi mavjud, markazda cart recap, payment card va summary card joylashgan, pastda esa Place Order tugmasi bo'lgan sticky submit bar turadi. Payment picker orqali Click, Payme, Global Pay yoki Cash on Delivery tanlanadi; muvaffaqiyatdan so'ng esa butun sahifa success-state bilan almashtiriladi.

Bu ekranning patenti uchun eng muhim jihat uning himoyalangan submit logikasidir. Agar supplier yopiq bo'lsa, interposed confirmation chiqadi. Agar yuborish xato bo'lsa, buyurtma PendingOrder sifatida saqlanadi va qayta urinishga tayyor turadi. Agar muvaffaqiyatli bo'lsa, cart tozalanadi va success holati beriladi. Demak checkout bitta submit emas, balki supplier availability, gateway mapping, retry safety va final order creation'ni birlashtiradigan boshqariladigan transition gate hisoblanadi.

**8. Driver iOS: Telefon Va PIN Orqali Kirish**
Asosiy dossye: patent-dossier/page-dossiers/driver-ios-secondary-surfaces.json
Yuza: ios-driver-login

Haydovchi kirish ekrani sodda ko'rinsa-da, juda aniq rolga moslangan. Yuqorida brand crest va sarlavha, keyin telefon maydoni, PIN maydoni, PIN visibility toggle va Login tugmasi joylashgan. Bunday minimal struktura ilovaning umumiy consumer app emas, balki marshrut va ijro operatsiyalari uchun ajratilgan protected shell ekanini ko'rsatadi.

Patent qiymati rolga xos autentifikatsiyada namoyon bo'ladi. Telefon plus PIN modeli, maxfiylikni boshqarish va muvaffaqiyatdan keyin to'g'ridan-to'g'ri himoyalangan shell'ga o'tish har bir rol uchun alohida entry figure mavjudligini isbotlaydi. Bu sahifa supplier registration va retailer checkout bilan qiyoslaganda, haydovchi uchun maxsus, qisqa va tezkor identifikatsiya rejimini ko'rsatadi.

**9. Driver iOS: Ijro Xarita Yuzasi**
Asosiy dossye: patent-dossier/page-dossiers/driver-ios-secondary-surfaces.json
Yuza: ios-driver-map

Driver map butun execution zanjirining markaziy tugunidir. Bu yerda live mission marker'lar, Me or Target or Both focus control, selected mission detail pane va Scan QR hamda Correct Delivery amallari bir joyda birlashgan. Driver aynan shu xaritadan scan oqimiga o'tadi, keyin offload review, payment yoki cash collection, kerak bo'lsa correction yo'liga kiradi.

Huquqiy jihatdan bu yuzaning kuchi shundaki, route, telemetry, geofence, task selection va settlement branching alohida ekranlarga parchalanmaydi. Bularning barchasi bitta operational map surface ichida jamlangan. Shu sababli ushbu ekran dispatch spine, telemetry loop va delivery execution chain bo'yicha kuchli figure material hisoblanadi.

**10. Driver Android: Offload Review**
Asosiy dossye: patent-dossier/page-dossiers/driver-android-offload-review.json

Offload review yetkazib berishning fizik holatini buyurtmaning moliyaviy natijasiga aylantiradigan birinchi rasmiy bosqichdir. Yuqorida retailer identifikatori, pastroqda original va adjusted totals, markazda esa line item ro'yxati, status ikonalar va rejected quantity stepper'lari berilgan. Pastda Confirm Offload yoki Amend and Confirm Offload tugmasi joylashadi.

Patent mazmuni shundaki, bu ekran oddiy unload tasdig'i emas. Driver har bir satr bo'yicha yetib kelmagan yoki shikastlangan birliklarni chiqarishi mumkin, tizim esa shu zahoti summani qayta hisoblaydi. Keyingi payment yoki cash oqimi aynan shu corrected offload result asosida davom etadi. Demak bu yuzada fizik offload bilan settlement truth o'rtasidagi isbotlanadigan bog'lanish yuzaga chiqadi.

**11. Driver Android: Naqd Pul Yig'ish**
Asosiy dossye: patent-dossier/page-dossiers/driver-android-cash-collection.json

Cash collection ekrani butun diqqatni bitta kritik harakatga qaratadi: naqd to'lov qabul qilinganini ochiq tasdiqlash. Center-stack ichida Payments ikonkasi, COLLECT CASH sarlavhasi, summa va tushuntiruvchi matn turadi, pastda esa Cash Collected — Complete tugmasi bor. Agar driver ortga chiqmoqchi bo'lsa, maxsus confirm dialog ko'rsatiladi.

Patent uchun bu muhim, chunki cash settlement delivery completion ichiga singdirib yuborilmagan. U alohida, himoyalangan va back-navigation bilan buzib bo'lmaydigan gate sifatida ajratilgan. Natijada naqd pul qabul qilingani audit-safe tarzda qayd etiladi va delivery closure nazorat ostida yakunlanadi.

**12. Driver Android: Delivery Correction**
Asosiy dossye: patent-dossier/page-dossiers/driver-android-delivery-correction.json

Delivery correction ekrani offload review'ni to'liq amendment workflow'ga aylantiradi. Yuqorida Verify Cargo va modified-count badge turadi, markazda manifest line item'lari ko'rsatiladi, tanlangan element uchun esa bottom sheet ochilib accepted quantity, rejected quantity, reason chip'lar va adjusted line total preview boshqariladi. Pastdagi sticky footer original total, refund delta va adjusted total'ni doim ko'rsatib turadi.

Bu sahifaning patent ahamiyati shundaki, line-level amendment, reason coding va refund preview bir xil izchil foydalanuvchi oqimiga bog'langan. Driver Submit Amendment tugmasini bosganda, tizimga oddiy izoh emas, balki manifestning yangi operatsion va moliyaviy versiyasi yuboriladi. Bu bounded human override va machine-readable correction model'ining to'g'ridan-to'g'ri namunasidir.

**13. Payload Terminal: Manifest Ish Maydoni**
Asosiy dossye: patent-dossier/page-dossiers/payload-manifest-workspace.json

Payload terminalning ikki panelli manifest workspace'i ombor tayyorgarligini dispatch release bilan birlashtiradi. Chapda truck va order ro'yxati, o'ngda esa tanlangan order sarlavhasi, retailer identifikatori, payment gateway, summa va checklist joylashgan. Har bir checklist satri to'liq-row toggle sifatida ishlaydi, Mark as Loaded esa faqat barcha item'lar tekshirilgandan keyin faollashadi.

Patent bo'yicha bu yuzaning kuchi shundaki, loading odatiy ombor amali sifatida emas, formal mashina hodisasi sifatida qayd etiladi. /v1/payload/seal chaqiruvidan keyin order sealedOrderIds ichiga o'tadi va dispatch uchun tayyor deb hisoblanadi. Shunday qilib, ekran warehouse labor'ni keyingi route automation uchun machine-readable seal event'ga aylantiradi.

**14. Payload Terminal: Dispatch Success**
Asosiy dossye: patent-dossier/page-dossiers/payload-dispatch-success.json

Dispatch success holati ombor oqimining yakuniy natijasini aniq ko'rsatadi. Markazda active truck, Manifest Secured va Fleet Dispatched matnlari turadi. Agar dispatch code'lar mavjud bo'lsa, alohida panel order ID dan code ga juftliklar ko'rinishida chiqadi. Pastda New Manifest tugmasi terminalni yangi ish sikliga qaytaradi.

Huquqiy nuqtai nazardan bu ekran seal holatining tashqi, kuzatiladigan oqibatini ko'rsatadi. Dispatch code paneli ombor handoff'ini visual va machine-referencable holatga keltiradi, New Manifest esa nazoratli reset transition'ni isbotlaydi. Shu sababli bu yuzani manifest handshake claim family bo'yicha yakuniy dalil yuzasi deb ko'rish mumkin.

**Yakun**
Tanlangan yuzalar oltita asosiy patent o'qini yopadi: role-specific entry, supplier intelligence, product va settlement governance, retailer commerce intent capture, driver execution with payment branching, hamda warehouse manifest sealing. Bu review paket inglizcha JSON dossyelar va figure-production-catalog bilan birga ishlatilganda, asosiy tezisni ushlab turadi: bu yerda himoya qilinayotgan narsa parchalangan ilovalar emas, balki boshqariladigan state zanjiridir.
