define config.developer = False
define config.steam_appid = 4050270

define config.default_music_volume = .8
default preferences.volume.ambient = .7

init:
    $ day_1 = 0




init python:

    old_replace_text = config.replace_text

    def fix_dashes_text(s):
        # Если уже был свой replace_text — сначала применяем его
        if old_replace_text:
            s = old_replace_text(s)

        # Меняем только минус с пробелами вокруг.
        # Слова типа "что-то", "кое-как", "по-прежнему" не трогаем.
        s = s.replace(" - ", " — ")

        return s

    config.replace_text = fix_dashes_text



transform in2left(x=config.screen_width, t=.5):
    subpixel True
    xoffset x
    ease t xoffset 0

transform in2right(x=-config.screen_width, t=.5):
    subpixel True
    xoffset x
    ease t xoffset 0

screen skip_splash:
    key "dismiss" action Jump("skip_splash")

label splashscreen:
    if persistent.lang is None:
        scene expression "#023" with None

        menu:
            "English":
                $ Language("english")()

            "Русский":
                $ Language(None)()

        $ persistent.lang = True

        show screen skip_splash
        scene black with Dissolve(.2)

    else:
        show screen skip_splash
        scene black with None

    show logo1 at in2right(t=.75):
        2.75
        ease .5 alpha 0

    show logo2 at in2left(t=.75):
        2.75
        ease .5 alpha 0

    pause .5

    $ splay("ding")

    pause 2

    scene black
    with dissolve

    show text _("Все персонажи в игре взрослые и им более 18 лет.{w=2}{nw}")
    with dissolve

    pause 2

    scene black
    with dissolve

label skip_splash:
    hide screen skip_splash

    return



label start:
    $ _skipping = True

    # на старте будет доступен сектор южнее замка в левом верхнем углу
    $ set_world_map( 2, 2 )

    # создание персонажа, которого будем прокачивать
    $ b_girl = Pers(_("Селена"), kind="hero", h=260)

    # создание команды наших
    $ b_heroes = Party( b_girl, enemy=False )

    # запускаем пролог
    #jump prologue

    # тестируем разные вещи
    if config.developer:
        menu:
            "Пролог":
                jump prologue

            "Глава 1":
                jump glava_1

            "Кабинет":
                call add_events from _call_add_events_1
                jump kabinet

            "Тест спрайта":
                $ mstop()
                $ ShowMenu("gg_test")()
                jump start

            "Тест боёвки":
                $ b_enemies = Party( Pers(_("Разбойник 1"), spr="bandit", hp=25), Pers(_("Разбойник 2"), spr="bandit", hp=25), Pers(_("Разбойник 3"), spr="bandit", hp=25) )
                $ b_heroes = Party( b_girl, Pers(_("Вампирша"), kind="friend", h=260), enemy=False )

                call battle from _call_battle

    jump prologue
