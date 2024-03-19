from django.db import models
from django.contrib.auth.models import AbstractUser
from django.utils import timezone

class django_user(AbstractUser):
    user = models.ForeignKey('user', blank=True, null=True, on_delete=models.CASCADE)

    class Meta:
        verbose_name_plural = "Django Users"
        
class user(models.Model):
    id = models.CharField(max_length=255,primary_key=True)
    name = models.CharField(max_length=255, blank=True, null=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField()

    def save(self, *args, **kwargs):
            self.updated_at = timezone.now()
            super(user, self).save(*args, **kwargs)

    def __str__(self):
        return f"{self.name}"


class playlist(models.Model):
    user = models.ForeignKey(user, blank=True, null=True, on_delete=models.CASCADE)
    id = models.CharField(max_length=255,primary_key=True)
    title = models.CharField(max_length=255, blank=True, null=True)
    description = models.CharField(max_length=255, blank=True, null=True) 
    thumbnail = models.CharField(max_length=255, blank=True, null=True)
    is_private = models.BooleanField(blank=True, null=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField()
    last_refresh = models.DateTimeField(blank=True, null=True)

    def save(self, *args, **kwargs):
            self.updated_at = timezone.now()
            super(playlist, self).save(*args, **kwargs)

    def __str__(self):
        return f"{self.title}"
    
class video(models.Model):
    id = models.CharField(max_length=255,primary_key=True)
    title = models.CharField(max_length=255, blank=True, null=True)
    description = models.CharField(max_length=255, blank=True, null=True)
    thumbnail = models.CharField(max_length=255, blank=True, null=True) 
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField()

    def save(self, *args, **kwargs):
            self.updated_at = timezone.now()
            super(video, self).save(*args, **kwargs)

    def __str__(self):
        return f"{self.title}"
    
class playlist_video(models.Model):
        playlist = models.ForeignKey(playlist, on_delete=models.CASCADE)
        video = models.ForeignKey(video, on_delete=models.CASCADE)