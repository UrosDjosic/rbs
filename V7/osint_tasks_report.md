# OSINT zadaci write up

## 2. PIRATES OF MEMORIAL

**Zadatak:** Pronađi flag u komentaru originalnog fotografa.

**Odgovor:** `csictf{pl4g14r1sm_1s_b4d}`

Sliku sam pretražio pomoću Google Lensa. U rezultatima sam našao [tweet](https://x.com/vivbajaj/status/1263046172282949632) gde je objavljena ista fotografija Victoria Memoriala u Kalkuti. U komentarima se pominje pravi autor, [Arunopal Banerjee](https://x.com/arunopal17).

![Google Lens pretraga](./osint_images/task2image1.png)

![Tweet sa fotografijom](./osint_images/task2image2.png)

Na X profilu u komentarima nisam našao flag, pa sam prešao na njegov Instagram. Na [originalnom postu](https://www.instagram.com/p/B3oKrLQgpko/) flag se nalazi u komentarima.

![X profil fotografa](./osint_images/task2image3.png)

![Flag u Instagram komentarima](./osint_images/task2image4.png)

## 3. COMMITMENT

**Zadatak:** Prati korisnika `hoshimaseok` i pronađi flag.

**Odgovor:** `csictf{sc4r3d_0f_c0mm1tm3nt}`

Username sam ukucao u Google i došao do [GitHub profila](https://github.com/hoshimaseok). Pregledao sam repozitorijume i repozitorijum SomethingFishy mi je delovao sumnjivo, pogotovo jer naziv zadatka Commitment ukazuje na git commit istoriju.

Prvo sam pretražio fajlove u repou upisujući `csictf` u GitHub Code search, ali nisam našao ništa. Shvatio sam da flag nije u trenutnom stanju koda, već u istoriji commitova. Otišao sam na Commits, prebacio se na granu `dev` jer ima mnogo više commitova od `master` grane, i počeo da pregledam commitove jedan po jedan. Za svaki commit sam otvarao Files changed i gledao da li ima obrisanih fajlova, koji su označeni crvenom bojom.

![Commit istorija na grani dev](./osint_images/task3image1.png)

U commitu `fix: userSchema.js fix` pronašao sam obrisan fajl `.env` koji je sadržao flag:

https://github.com/hoshimaseok/SomethingFishy/commit/5d3ea213285ba9afab22b6f8f3a3faddabbd1cae

![Obrisan .env fajl sa flagom](./osint_images/task3image2.png)
