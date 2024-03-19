from django.contrib import admin
from django.contrib.auth.admin import UserAdmin
from . import models

class DjangoUserAdmin(admin.ModelAdmin):
    list_filter = UserAdmin.list_filter
    fieldsets = UserAdmin.fieldsets
    list_display = UserAdmin.list_display

class UserAdmin(admin.ModelAdmin):
    search_fields=['id','name',]
    list_display=('id','name','created_at','updated_at')

class PlaylistAdmin(admin.ModelAdmin):
    raw_id_fields = ["user"]
    search_fields=['id','title',]
    list_display=('id','title','description','thumbnail','is_private','created_at','updated_at','last_refresh')

class VideoAdmin(admin.ModelAdmin):
    search_fields=['id','title',]
    list_display=('id','title','description','thumbnail','created_at','updated_at')

class PlaylistVideoAdmin(admin.ModelAdmin):
    search_fields=['video','playlist']
    raw_id_fields=['video','playlist']

admin.site.register(models.playlist_video,PlaylistVideoAdmin)
admin.site.register(models.video, VideoAdmin)
admin.site.register(models.playlist, PlaylistAdmin)
admin.site.register(models.django_user, DjangoUserAdmin)
admin.site.register(models.user, UserAdmin)
